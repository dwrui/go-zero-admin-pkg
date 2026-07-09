package excel

import "testing"

func TestEnrichDicColumnsFixTextType(t *testing.T) {
	mapping := &ImportMapping{
		Columns: []ImportColumn{
			{
				ExcelHeader: "工厂类型",
				DbField:     "plant_type",
				FieldType:   "text",
				DicGroupID:  1,
				OptionValue: "1=生产工厂,2=贸易物流中心",
			},
		},
	}
	EnrichDicColumns(mapping)
	col := mapping.Columns[0]
	if col.FieldType != "belongDic" {
		t.Fatalf("field_type want belongDic got %s", col.FieldType)
	}
	if col.OnNotFound != "error" {
		t.Fatalf("on_not_found want error got %s", col.OnNotFound)
	}
}

func TestMergeSystemImportColumnsPlantType(t *testing.T) {
	tenant := &ImportMapping{
		Columns: []ImportColumn{
			{
				ExcelHeader:    "工厂类型",
				DbField:        "plant_type",
				FieldType:      "text",
				Importable:     true,
				TenantEditable: false,
			},
		},
	}
	systemRaw := `{"version":1,"columns":[{"db_field":"plant_type","field_type":"belongDic","dic_group_id":1,"option_value":"1=生产工厂,2=贸易物流中心","tenant_editable":false,"importable":true}]}`
	if err := MergeSystemImportColumns(tenant, systemRaw); err != nil {
		t.Fatal(err)
	}
	col := tenant.Columns[0]
	if col.FieldType != "belongDic" || col.DicGroupID != 1 {
		t.Fatalf("got type=%s dic_group_id=%d", col.FieldType, col.DicGroupID)
	}
}

func TestMergeSystemImportColumnsIsDefault(t *testing.T) {
	tenant := &ImportMapping{
		Columns: []ImportColumn{
			{
				ExcelHeader:    "是否默认工厂",
				DbField:        "is_default",
				FieldType:      "text",
				Importable:     true,
				TenantEditable: true,
			},
		},
	}
	systemRaw := `{"version":1,"columns":[{"db_field":"is_default","field_type":"radio","option_value":"0=否,1=是","tenant_editable":true,"importable":true}]}`
	if err := MergeSystemImportColumns(tenant, systemRaw); err != nil {
		t.Fatal(err)
	}
	col := tenant.Columns[0]
	if col.FieldType != "radio" || col.OptionValue != "0=否,1=是" {
		t.Fatalf("got type=%s option=%q", col.FieldType, col.OptionValue)
	}
	v, err := ConvertImportCell(col, "是", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := v.(int64); !ok || n != 1 {
		t.Fatalf("want 1 got %v", v)
	}
}

func TestConvertImportCellRadioIsDefault(t *testing.T) {
	col := ImportColumn{
		ExcelHeader: "是否默认工厂", DbField: "is_default", FieldType: "radio",
		OptionValue: "0=否,1=是",
	}
	v, err := ConvertImportCell(col, "是", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := v.(int64); !ok || n != 1 {
		t.Fatalf("want int64 1 got %v", v)
	}
}

func TestNormalizeImportFieldTypeDic(t *testing.T) {
	if got := normalizeImportFieldType("dic"); got != "belongDic" {
		t.Fatalf("dic want belongDic got %s", got)
	}
	if got := normalizeImportFieldType("belongDic"); got != "belongDic" {
		t.Fatalf("belongDic want belongDic got %s", got)
	}
}
