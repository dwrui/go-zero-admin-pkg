package excel

import "testing"

func TestExportableColumnsDefaultTrue(t *testing.T) {
	mapping, err := ParseExportMapping(`{
		"version": 1,
		"columns": [
			{"sort": 1, "excel_header": "工厂名称", "db_field": "plant_name", "field_type": "text"},
			{"sort": 2, "excel_header": "ID", "db_field": "id", "field_type": "number", "exportable": false}
		]
	}`)
	if err != nil {
		t.Fatal(err)
	}
	cols := mapping.ExportableColumns()
	if len(cols) != 1 {
		t.Fatalf("expected 1 exportable column, got %d", len(cols))
	}
	if cols[0].DbField != "plant_name" {
		t.Fatalf("unexpected column: %s", cols[0].DbField)
	}
}

func TestColumnConfigsUseExcelHeader(t *testing.T) {
	mapping, err := ParseExportMapping(`{
		"version": 1,
		"columns": [
			{"sort": 1, "excel_header": "创建时间", "db_field": "create_time", "field_type": "time", "exportable": true}
		]
	}`)
	if err != nil {
		t.Fatal(err)
	}
	cols := mapping.ColumnConfigs()
	if len(cols) != 1 || cols[0].Title != "创建时间" || cols[0].Field != "create_time" {
		t.Fatalf("unexpected configs: %+v", cols)
	}
}
