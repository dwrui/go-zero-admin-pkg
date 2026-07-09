package excel

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestExportExcelRegionByCode(t *testing.T) {
	fieldConfigs := []FieldConfig{
		{Field: "province_code", NameField: "province_codeName", Datatable: "ga_base_region", Datatablename: "name", Formtype: "region"},
		{Field: "plant_type", OptionValue: "1=生产工厂,2=贸易物流中心", Formtype: "belongDic", DicGroupId: 1},
	}
	columns := []ColumnConfig{
		{Title: "省份", Field: "province_codeName"},
		{Title: "类型", Field: "plant_type"},
	}
	type row struct {
		ProvinceCode int64
		PlantType    int64
	}
	list := []row{{ProvinceCode: 110000, PlantType: 1}}
	relFn := func(tableName, keyField, valueField, code string) string {
		if tableName == "ga_base_region" && keyField == "region_code" && code == "110000" {
			return "北京"
		}
		return ""
	}
	dicFn := func(groupId int64, keyvalue string) string {
		if groupId == 1 && keyvalue == "1" {
			return "生产工厂"
		}
		return keyvalue
	}
	data, err := ExportExcel(columns, fieldConfigs, list, nil, dicFn, relFn)
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	v, err := f.GetCellValue("Sheet1", "A2")
	if err != nil || v != "北京" {
		t.Fatalf("province want 北京 got %q err %v", v, err)
	}
	v, err = f.GetCellValue("Sheet1", "B2")
	if err != nil || v != "生产工厂" {
		t.Fatalf("plant_type want 生产工厂 got %q err %v", v, err)
	}
}
