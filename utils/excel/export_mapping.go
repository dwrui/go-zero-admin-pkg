package excel

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ExportMapping 导出模板 field_mapping JSON（version=1）
type ExportMapping struct {
	Version     int                `json:"version"`
	TargetTable string             `json:"target_table"`
	BizType     string             `json:"biz_type"`
	Columns     []ExportColumn     `json:"columns"`
	Meta        ImportMappingMeta  `json:"meta,omitempty"`
}

// ExportColumn 导出列
type ExportColumn struct {
	Sort           int    `json:"sort"`
	ExcelHeader    string `json:"excel_header"`
	DbField        string `json:"db_field"`
	FieldType      string `json:"field_type"`
	Exportable     bool   `json:"exportable"`
	TenantEditable bool   `json:"tenant_editable"`
	OptionValue    string `json:"option_value,omitempty"`
	RefTable       string `json:"ref_table,omitempty"`
	RefDisplayField string `json:"ref_display_field,omitempty"`
	DicGroupID     int64  `json:"dic_group_id,omitempty"`
	Width          int    `json:"width,omitempty"`
}

// ExportColumnConfig 对接各模块 excel.ExportExcel 的列配置
type ExportColumnConfig struct {
	Title string
	Field string
}

// DefaultExportTplCode 导出默认模板编码：base_brand_export_default
func DefaultExportTplCode(tableName string) string {
	return fmt.Sprintf("%s_export_default", strings.TrimPrefix(tableName, "ga_"))
}

// DefaultImportTplCode 导入默认模板编码：base_brand_default
func DefaultImportTplCode(tableName string) string {
	return fmt.Sprintf("%s_default", strings.TrimPrefix(tableName, "ga_"))
}

// BuildExportMapping 从列表字段生成系统默认导出 mapping
func BuildExportMapping(bizType, targetTable, moduleName string, generateCodeID int64, fields []GenerateCodeField) (*ExportMapping, error) {
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

	mapping := &ExportMapping{
		Version:     1,
		TargetTable: targetTable,
		BizType:     bizType,
		Meta: ImportMappingMeta{
			ModuleName:   moduleName,
			GenerateCode: generateCodeID,
		},
	}
	sortOrder := 1
	for _, f := range sorted {
		if f.Islist != 1 {
			continue
		}
		if f.Field == "operations" {
			continue
		}
		col := ExportColumn{
			Sort:           sortOrder,
			ExcelHeader:    f.Name,
			DbField:        f.Field,
			FieldType:      normalizeImportFieldType(f.Formtype),
			Exportable:     true,
			TenantEditable: isTenantEditableFieldType(f.Formtype),
			OptionValue:    f.OptionValue,
		}
		if f.Formtype == "belongto" {
			col.RefTable = f.Datatable
			col.RefDisplayField = f.Datatablename
		}
		if f.Formtype == "belongDic" {
			col.DicGroupID = f.DicGroupID
		}
		mapping.Columns = append(mapping.Columns, col)
		sortOrder++
	}
	if len(mapping.Columns) == 0 {
		return nil, fmt.Errorf("no exportable columns for %s", bizType)
	}
	return mapping, nil
}

func (m *ExportMapping) JSON() (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ParseExportMapping(raw string) (*ExportMapping, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("empty field_mapping")
	}
	var m ExportMapping
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	if m.Version == 0 {
		m.Version = 1
	}
	return &m, nil
}

func (m *ExportMapping) ExportableColumns() []ExportColumn {
	out := make([]ExportColumn, 0, len(m.Columns))
	for _, c := range m.Columns {
		if c.Exportable {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Sort < out[j].Sort })
	return out
}

func (m *ExportMapping) ColumnConfigs() []ExportColumnConfig {
	cols := m.ExportableColumns()
	out := make([]ExportColumnConfig, 0, len(cols))
	for _, c := range cols {
		title := c.ExcelHeader
		if title == "" {
			title = c.DbField
		}
		out = append(out, ExportColumnConfig{Title: title, Field: c.DbField})
	}
	return out
}
