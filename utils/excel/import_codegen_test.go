package excel

import "testing"

func TestEnrichColumnsFromCodegenIsDefault(t *testing.T) {
	mapping := &ImportMapping{
		Columns: []ImportColumn{
			{DbField: "is_default", FieldType: "radio", ExcelHeader: "是否默认工厂", Importable: true},
			{DbField: "status", FieldType: "radio", ExcelHeader: "状态", Importable: true},
		},
	}
	fields := []CodegenFieldMeta{
		{Field: "is_default", Formtype: "radio", OptionValue: "0=否,1=是"},
		{Field: "status", Formtype: "radio", OptionValue: "0=禁用,1=正常"},
	}
	EnrichColumnsFromCodegen(mapping, fields)
	if mapping.Columns[0].OptionValue != "0=否,1=是" {
		t.Fatalf("is_default option %q", mapping.Columns[0].OptionValue)
	}
	v, err := ConvertImportCell(mapping.Columns[0], "是", nil, nil)
	if err != nil || v.(int64) != 1 {
		t.Fatalf("is_default 是 -> %v err %v", v, err)
	}
}

func TestOptionValueFromColumnComment(t *testing.T) {
	got := OptionValueFromColumnComment("是否默认工厂:0=否,1=是")
	if got != "0=否,1=是" {
		t.Fatalf("got %q", got)
	}
}

func TestEnrichColumnsFromTableComments(t *testing.T) {
	mapping := &ImportMapping{
		Columns: []ImportColumn{
			{DbField: "status", FieldType: "text", ExcelHeader: "状态", Importable: true},
		},
	}
	EnrichColumnsFromTableComments(mapping, map[string]string{
		"status": "状态:0=禁用,1=正常",
	})
	if mapping.Columns[0].OptionValue != "0=禁用,1=正常" {
		t.Fatalf("got %q", mapping.Columns[0].OptionValue)
	}
}
