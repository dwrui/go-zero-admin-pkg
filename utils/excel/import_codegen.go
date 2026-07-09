package excel

import "strings"

// CodegenFieldMeta 代码生成器字段元数据（用于导入/导出模板补全 option_value 等）
type CodegenFieldMeta struct {
	Field         string
	Name          string
	Formtype      string
	Searchtype    string
	OptionValue   string
	DicGroupID    int64
	Datatable     string
	Datatablename string
	IsSensitive   int64
	ValidateRules string
}

// EnrichColumnsFromCodegen 从 ga_common_generatecode_field 补全模板列的 option_value / field_type
func EnrichColumnsFromCodegen(mapping *ImportMapping, fields []CodegenFieldMeta) {
	if mapping == nil || len(fields) == 0 {
		return
	}
	byField := make(map[string]CodegenFieldMeta, len(fields))
	for _, f := range fields {
		if f.Field == "" {
			continue
		}
		byField[f.Field] = f
	}
	for i, col := range mapping.Columns {
		gf, ok := byField[col.DbField]
		if !ok {
			continue
		}
		mergeCodegenColumnMeta(&mapping.Columns[i], gf)
	}
	EnrichRegionColumns(mapping)
	EnrichDicColumns(mapping)
}

func mergeCodegenColumnMeta(col *ImportColumn, gf CodegenFieldMeta) {
	gen := GenerateCodeField{
		Field:         gf.Field,
		Formtype:      gf.Formtype,
		Searchtype:    gf.Searchtype,
		OptionValue:   gf.OptionValue,
		DicGroupID:    gf.DicGroupID,
		Datatable:     gf.Datatable,
		Datatablename: gf.Datatablename,
		Isform:        1,
	}
	resolvedType := resolveRegionImportFieldType(gen)

	if strings.TrimSpace(col.OptionValue) == "" && strings.TrimSpace(gf.OptionValue) != "" {
		col.OptionValue = strings.TrimSpace(gf.OptionValue)
	}
	if col.DicGroupID <= 0 && gf.DicGroupID > 0 {
		col.DicGroupID = gf.DicGroupID
	}
	if strings.EqualFold(gf.Formtype, "belongDic") || gf.DicGroupID > 0 {
		col.FieldType = "belongDic"
		col.DicGroupID = gf.DicGroupID
		if col.OnNotFound == "" {
			col.OnNotFound = "error"
		}
		return
	}
	if isEnumImportFieldType(resolvedType) && (isPlainImportFieldType(col.FieldType) || col.FieldType == "text") {
		col.FieldType = resolvedType
	}
	if isEnumImportFieldType(col.FieldType) && col.OnNotFound == "" {
		col.OnNotFound = "error"
	}
	if gf.Formtype == "belongto" && col.RefTable == "" {
		col.RefTable = gf.Datatable
		col.RefDisplayField = gf.Datatablename
	}
}

// OptionValueFromColumnComment 从字段 COMMENT 解析选项，如 状态:0=禁用,1=正常
func OptionValueFromColumnComment(comment string) string {
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return ""
	}
	idx := strings.Index(comment, ":")
	if idx < 0 {
		return ""
	}
	tail := strings.TrimSpace(comment[idx+1:])
	if !hasEnumOptionValue(tail) {
		return ""
	}
	return tail
}

// EnrichColumnsFromTableComments 用表字段 COMMENT 补全缺失的 option_value
func EnrichColumnsFromTableComments(mapping *ImportMapping, comments map[string]string) {
	if mapping == nil || len(comments) == 0 {
		return
	}
	for i, col := range mapping.Columns {
		if strings.TrimSpace(col.OptionValue) != "" {
			continue
		}
		if opt := OptionValueFromColumnComment(comments[col.DbField]); opt != "" {
			mapping.Columns[i].OptionValue = opt
			if isPlainImportFieldType(col.FieldType) {
				mapping.Columns[i].FieldType = "radio"
			}
			if mapping.Columns[i].OnNotFound == "" {
				mapping.Columns[i].OnNotFound = "error"
			}
		}
	}
	EnrichDicColumns(mapping)
}

// EnrichExportColumnsFromCodegen 从代码生成器补全导出模板列元数据
func EnrichExportColumnsFromCodegen(mapping *ExportMapping, fields []CodegenFieldMeta) {
	if mapping == nil || len(fields) == 0 {
		return
	}
	byField := make(map[string]CodegenFieldMeta, len(fields))
	for _, f := range fields {
		if f.Field == "" {
			continue
		}
		byField[f.Field] = f
	}
	for i := range mapping.Columns {
		gf, ok := byField[mapping.Columns[i].DbField]
		if !ok {
			continue
		}
		enrichExportColumnFromCodegen(&mapping.Columns[i], gf)
	}
}

func enrichExportColumnFromCodegen(col *ExportColumn, gf CodegenFieldMeta) {
	if strings.TrimSpace(col.ExcelHeader) == "" && strings.TrimSpace(gf.Name) != "" {
		col.ExcelHeader = strings.TrimSpace(gf.Name)
	}
	gen := GenerateCodeField{
		Field:         gf.Field,
		Formtype:      gf.Formtype,
		Searchtype:    gf.Searchtype,
		OptionValue:   gf.OptionValue,
		DicGroupID:    gf.DicGroupID,
		Datatable:     gf.Datatable,
		Datatablename: gf.Datatablename,
		Isform:        1,
	}
	resolvedType := resolveRegionImportFieldType(gen)
	if strings.TrimSpace(col.OptionValue) == "" && strings.TrimSpace(gf.OptionValue) != "" {
		col.OptionValue = strings.TrimSpace(gf.OptionValue)
	}
	if col.DicGroupID <= 0 && gf.DicGroupID > 0 {
		col.DicGroupID = gf.DicGroupID
	}
	if strings.EqualFold(gf.Formtype, "belongDic") || gf.DicGroupID > 0 {
		col.FieldType = "belongDic"
		col.TenantEditable = false
		return
	}
	if gf.Formtype == "belongto" {
		col.FieldType = "belongto"
		col.RefTable = gf.Datatable
		col.RefDisplayField = gf.Datatablename
		col.TenantEditable = false
		return
	}
	if isEnumImportFieldType(resolvedType) && (isPlainImportFieldType(col.FieldType) || col.FieldType == "" || col.FieldType == "text") {
		col.FieldType = resolvedType
	}
	if col.FieldType == "" || col.FieldType == "text" {
		switch strings.ToLower(strings.TrimSpace(gf.Formtype)) {
		case "time":
			col.FieldType = "time"
		case "date":
			col.FieldType = "date"
		case "datetime":
			col.FieldType = "datetime"
		}
	}
	if !col.Sensitive && (gf.IsSensitive == 1 || SuggestIsSensitive(gf.Field, gf.Name)) {
		col.Sensitive = true
		if strings.TrimSpace(col.MaskMode) == "" {
			col.MaskMode = DefaultMaskMode(gf.Field)
		}
	}
}
