package excel

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ImportMapping 导入模板 field_mapping JSON 根结构（version=1）
type ImportMapping struct {
	Version      int              `json:"version"`
	TargetTable  string           `json:"target_table"`
	BizType      string           `json:"biz_type"`
	DuplicateKey []string         `json:"duplicate_key,omitempty"`
	ImportMode   string           `json:"import_mode,omitempty"` // insert | upsert | update
	Columns      []ImportColumn   `json:"columns"`
	Meta         ImportMappingMeta `json:"meta,omitempty"`
}

type ImportMappingMeta struct {
	ModuleName   string `json:"module_name,omitempty"`
	GenerateCode int64  `json:"generatecode_id,omitempty"`
}

// ImportColumn 单列映射
type ImportColumn struct {
	Sort             int      `json:"sort"`
	ExcelHeader      string   `json:"excel_header"`
	DbField          string   `json:"db_field"`
	FieldType        string   `json:"field_type"`
	Required         bool     `json:"required"`
	Importable       bool     `json:"importable"`
	TenantEditable   bool     `json:"tenant_editable"`
	OptionValue      string   `json:"option_value,omitempty"`
	RefTable         string   `json:"ref_table,omitempty"`
	RefMatchFields   []string `json:"ref_match_fields,omitempty"`
	RefDisplayField  string   `json:"ref_display_field,omitempty"`
	OnNotFound       string   `json:"on_not_found,omitempty"` // error | skip | auto_create
	DicGroupID       int64    `json:"dic_group_id,omitempty"`
	DefaultValue     string   `json:"default_value,omitempty"`
	Example          string   `json:"example,omitempty"`
}

// GenerateCodeField 代码生成器字段元数据（develop 传入）
type GenerateCodeField struct {
	Name          string
	Field         string
	Formtype      string
	Datatable     string
	Datatablename string
	DicGroupID    int64
	Required      int64
	Isform        int64
	Islist        int64
	OptionValue   string
	DefValue      string
	FieldWeigh    int64
}

var defaultImportSkipFields = map[string]struct{}{
	"id": {}, "business_id": {}, "dept_id": {},
	"create_by": {}, "update_by": {}, "create_time": {}, "update_time": {}, "delete_time": {},
	"workflow_instance_id": {},
}

var defaultImportSkipFormtypes = map[string]struct{}{
	"image": {}, "images": {}, "audio": {}, "file": {}, "files": {}, "colorpicker": {},
}

// ShouldSkipImportField 判断是否排除出默认导入列
func ShouldSkipImportField(field, formtype string) bool {
	if _, ok := defaultImportSkipFields[field]; ok {
		return true
	}
	if _, ok := defaultImportSkipFormtypes[formtype]; ok {
		return true
	}
	return false
}

// BuildImportMapping 从 generatecode 字段生成系统默认导入 mapping
func BuildImportMapping(bizType, targetTable, moduleName string, generateCodeID int64, fields []GenerateCodeField) (*ImportMapping, error) {
	if bizType == "" || targetTable == "" {
		return nil, fmt.Errorf("bizType and targetTable are required")
	}
	sorted := append([]GenerateCodeField(nil), fields...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].FieldWeigh == sorted[j].FieldWeigh {
			return sorted[i].Field < sorted[j].Field
		}
		return sorted[i].FieldWeigh < sorted[j].FieldWeigh
	})

	mapping := &ImportMapping{
		Version:     1,
		TargetTable: targetTable,
		BizType:     bizType,
		ImportMode:  "upsert",
		Meta: ImportMappingMeta{
			ModuleName:   moduleName,
			GenerateCode: generateCodeID,
		},
	}
	dupKey := guessDuplicateKey(sorted, targetTable)
	if len(dupKey) > 0 {
		mapping.DuplicateKey = dupKey
	}

	sortOrder := 1
	for _, f := range sorted {
		if f.Isform != 1 || ShouldSkipImportField(f.Field, f.Formtype) {
			continue
		}
		col := ImportColumn{
			Sort:           sortOrder,
			ExcelHeader:    f.Name,
			DbField:        f.Field,
			FieldType:      normalizeImportFieldType(f.Formtype),
			Required:       f.Required == 1,
			Importable:     true,
			TenantEditable: isTenantEditableFieldType(f.Formtype),
			OptionValue:    f.OptionValue,
			DefaultValue:   f.DefValue,
			Example:        buildColumnExample(f),
		}
		if f.Formtype == "belongto" {
			col.RefTable = f.Datatable
			col.RefDisplayField = f.Datatablename
			col.RefMatchFields = buildRefMatchFields(f.Datatable, f.Datatablename, f.Field)
			col.OnNotFound = "error"
		}
		if f.Formtype == "belongDic" {
			col.DicGroupID = f.DicGroupID
			col.OnNotFound = "error"
		}
		if f.Formtype == "radio" || f.Formtype == "select" || f.Formtype == "switch" {
			col.OnNotFound = "error"
		}
		mapping.Columns = append(mapping.Columns, col)
		sortOrder++
	}
	if len(mapping.Columns) == 0 {
		return nil, fmt.Errorf("no importable columns for %s", bizType)
	}
	return mapping, nil
}

func normalizeImportFieldType(formtype string) string {
	switch formtype {
	case "belongto", "belongDic", "radio", "select", "switch", "checkbox", "textarea",
		"text", "number", "float", "date", "datetime", "time", "region":
		return formtype
	default:
		return "text"
	}
}

func isTenantEditableFieldType(formtype string) bool {
	switch formtype {
	case "belongto", "belongDic":
		return false
	default:
		return true
	}
}

func buildRefMatchFields(datatable, displayField, dbField string) []string {
	fields := []string{}
	if displayField != "" {
		fields = append(fields, displayField)
	}
	codeField := strings.TrimSuffix(dbField, "_id") + "_code"
	if codeField != dbField && codeField != displayField {
		fields = append(fields, codeField)
	}
	if strings.HasSuffix(datatable, "_category") {
		fields = append(fields, "name")
	}
	seen := map[string]struct{}{}
	uniq := make([]string, 0, len(fields))
	for _, f := range fields {
		if f == "" {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		uniq = append(uniq, f)
	}
	return uniq
}

func guessDuplicateKey(fields []GenerateCodeField, table string) []string {
	candidates := []string{"code", "sn", "no"}
	fieldSet := map[string]struct{}{}
	for _, f := range fields {
		fieldSet[f.Field] = struct{}{}
	}
	for _, suffix := range candidates {
		for f := range fieldSet {
			if f == suffix || strings.HasSuffix(f, "_"+suffix) {
				return []string{f}
			}
		}
	}
	short := strings.TrimPrefix(table, "ga_base_")
	short = strings.TrimPrefix(short, "ga_")
	key := short + "_code"
	if _, ok := fieldSet[key]; ok {
		return []string{key}
	}
	return nil
}

func buildColumnExample(f GenerateCodeField) string {
	switch f.Formtype {
	case "belongto":
		if f.Datatablename != "" {
			return "示例" + f.Name
		}
	case "radio", "select", "switch":
		if f.OptionValue != "" {
			parts := strings.Split(f.OptionValue, ",")
			if len(parts) > 0 {
				kv := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
				if len(kv) == 2 {
					return kv[1]
				}
			}
		}
	case "number", "float":
		return "0"
	}
	return ""
}

func (m *ImportMapping) JSON() (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ParseImportMapping(raw string) (*ImportMapping, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("empty field_mapping")
	}
	var m ImportMapping
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	if m.Version == 0 {
		m.Version = 1
	}
	return &m, nil
}

// GenerateImportTemplateBytes 按 mapping 生成 xlsx（含说明行 + 示例行）
func GenerateImportTemplateBytes(mapping *ImportMapping, tplName string) ([]byte, error) {
	if mapping == nil {
		return nil, fmt.Errorf("mapping is nil")
	}
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Sheet1"
	_ = f.SetSheetName("Sheet1", sheet)

	cols := mapping.Columns
	sort.Slice(cols, func(i, j int) bool { return cols[i].Sort < cols[j].Sort })

	for i, col := range cols {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		header := col.ExcelHeader
		if col.Required {
			header += "*"
		}
		_ = f.SetCellValue(sheet, cell, header)
		noteCell, _ := excelize.CoordinatesToCellName(i+1, 2)
		_ = f.SetCellValue(sheet, noteCell, columnNote(col))
		if col.Example != "" {
			exCell, _ := excelize.CoordinatesToCellName(i+1, 3)
			_ = f.SetCellValue(sheet, exCell, col.Example)
		}
	}
	if tplName != "" {
		_ = f.SetCellValue(sheet, "A4", "模板:"+tplName)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func columnNote(col ImportColumn) string {
	switch col.FieldType {
	case "belongto":
		fields := strings.Join(col.RefMatchFields, "/")
		return fmt.Sprintf("关联%s，填%s", col.RefTable, fields)
	case "belongDic":
		if col.DicGroupID > 0 {
			return "字典项名称或键值"
		}
		if col.OptionValue != "" {
			return "可选:" + col.OptionValue
		}
		return "填选项值或标签"
	case "radio", "select", "switch":
		if col.OptionValue != "" {
			return "可选:" + col.OptionValue
		}
	}
	if col.Required {
		return "必填"
	}
	return ""
}

// ImportableColumns 返回启用导入的列
func (m *ImportMapping) ImportableColumns() []ImportColumn {
	out := make([]ImportColumn, 0, len(m.Columns))
	for _, c := range m.Columns {
		if c.Importable {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Sort < out[j].Sort })
	return out
}
