package excel

import (
	"fmt"
	"testing"
)

func TestResolveOptionValue(t *testing.T) {
	opt := "1=启用,0=禁用"
	v, err := resolveOptionValue(opt, "启用", "状态")
	if err != nil || v != "1" {
		t.Fatalf("label: got %q err %v", v, err)
	}
	v, err = resolveOptionValue(opt, "0", "状态")
	if err != nil || v != "0" {
		t.Fatalf("key: got %q err %v", v, err)
	}
	_, err = resolveOptionValue(opt, "未知", "状态")
	if err == nil {
		t.Fatal("expected error for invalid option")
	}
}

func TestConvertImportCellRequired(t *testing.T) {
	col := ImportColumn{ExcelHeader: "名称", DbField: "name", FieldType: "text", Required: true}
	_, err := ConvertImportCell(col, "", nil, nil)
	if err == nil {
		t.Fatal("expected required error")
	}
}

func TestConvertImportCellNumber(t *testing.T) {
	col := ImportColumn{ExcelHeader: "排序", DbField: "sort", FieldType: "number", Required: false}
	v, err := ConvertImportCell(col, "10", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v.(int64) != 10 {
		t.Fatalf("got %v", v)
	}
	_, err = ConvertImportCell(col, "abc", nil, nil)
	if err == nil {
		t.Fatal("expected number error")
	}
}

func TestParseImportRowsMixed(t *testing.T) {
	mapping := &ImportMapping{
		Version: 1,
		Columns: []ImportColumn{
			{Sort: 1, ExcelHeader: "品牌名称", DbField: "brand_name", FieldType: "text", Required: true, Importable: true},
			{Sort: 2, ExcelHeader: "状态", DbField: "status", FieldType: "radio", Required: false, Importable: true, OptionValue: "1=正常,0=禁用"},
		},
	}
	raw := []map[string]string{
		{"brand_name": "A", "status": "正常"},
		{"brand_name": "", "status": "1"},
		{"brand_name": "B", "status": "坏值"},
	}
	valid, errs := ParseImportRows(mapping, raw, []int{4, 5, 6}, nil, nil)
	if len(valid) != 1 {
		t.Fatalf("valid rows want 1 got %d", len(valid))
	}
	if valid[0]["status"] != "1" {
		t.Fatalf("status want 1 got %v", valid[0]["status"])
	}
	if len(errs) < 2 {
		t.Fatalf("expected at least 2 errors got %d", len(errs))
	}
}

func TestConvertImportCellBelongDicFallbackOption(t *testing.T) {
	col := ImportColumn{
		ExcelHeader: "状态", DbField: "status", FieldType: "belongDic",
		OptionValue: "1=启用,0=禁用",
	}
	v, err := ConvertImportCell(col, "启用", nil, nil)
	if err != nil || v != "1" {
		t.Fatalf("got %v err %v", v, err)
	}
}

type mockDicResolver struct {
	values map[string]string
}

func (m *mockDicResolver) ResolveBelongDic(col ImportColumn, displayValue string) (string, error) {
	if v, ok := m.values[displayValue]; ok {
		return v, nil
	}
	return "", fmt.Errorf("not found")
}

func TestConvertImportCellBelongDicWithGroup(t *testing.T) {
	col := ImportColumn{
		ExcelHeader: "工厂类型", DbField: "plant_type", FieldType: "belongDic",
		DicGroupID: 1,
	}
	resolver := &mockDicResolver{values: map[string]string{"生产工厂": "1", "1": "1"}}
	v, err := ConvertImportCell(col, "生产工厂", nil, resolver)
	if err != nil || v != "1" {
		t.Fatalf("got %v err %v", v, err)
	}
}

type mockBelongToResolver struct {
	id int64
}

func (m *mockBelongToResolver) ResolveBelongTo(col ImportColumn, displayValue string) (int64, error) {
	if displayValue == "品牌A" {
		return m.id, nil
	}
	return 0, fmt.Errorf("not found")
}

func TestConvertImportCellBelongTo(t *testing.T) {
	col := ImportColumn{
		ExcelHeader: "品牌", DbField: "brand_id", FieldType: "belongto",
		RefTable: "ga_goods_brand", RefMatchFields: []string{"brand_name"},
	}
	resolver := &mockBelongToResolver{id: 99}
	v, err := ConvertImportCell(col, "品牌A", resolver, nil)
	if err != nil || v.(int64) != 99 {
		t.Fatalf("got %v err %v", v, err)
	}
	_, err = ConvertImportCell(col, "不存在", resolver, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
