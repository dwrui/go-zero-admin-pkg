package excel

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestGenerateImportTemplateLayout(t *testing.T) {
	mapping := &ImportMapping{
		Columns: []ImportColumn{
			{Sort: 1, ExcelHeader: "工厂名称", DbField: "plant_name", Required: true},
			{Sort: 2, ExcelHeader: "省份", DbField: "province_code", FieldType: "region"},
		},
	}
	data, err := GenerateImportTemplateBytes(mapping, "工厂/Plant主数据导入模板")
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	h1, _ := f.GetCellValue("Sheet1", "A1")
	if h1 != "工厂名称*" {
		t.Fatalf("header want 工厂名称* got %q", h1)
	}
	meta, _ := f.GetCellValue("Sheet1", "A2")
	if !strings.Contains(meta, "模板:工厂/Plant主数据导入模板") {
		t.Fatalf("A2 meta want 模板:... got %q", meta)
	}
	if !strings.Contains(meta, "必填") {
		t.Fatalf("A2 should contain 必填 hint, got %q", meta)
	}
	b2, _ := f.GetCellValue("Sheet1", "B2")
	if b2 == "" {
		t.Fatalf("B2 should have region hint")
	}
	ex, _ := f.GetCellValue("Sheet1", "A3")
	if ex != "" {
		t.Fatalf("row3 should be empty, got %q", ex)
	}
}

func TestValidateImportTemplateLayout(t *testing.T) {
	valid := [][]string{
		{"工厂名称*", "省份"},
		{"模板:工厂导入模板；必填", "填省市区名称或地区码"},
		{"真实工厂", "江苏省"},
	}
	if err := ValidateImportTemplateLayout(valid); err != nil {
		t.Fatalf("valid template: %v", err)
	}
	if ImportTemplateDataStartIndex() != 2 {
		t.Fatalf("data start index want 2")
	}

	tooShort := [][]string{{"表头"}}
	if err := ValidateImportTemplateLayout(tooShort); err == nil {
		t.Fatal("expected error for too few rows")
	}
}
