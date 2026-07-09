package excel

import (
	"testing"
)

type mockRegionResolver struct {
	values map[string]int64
}

func (m *mockRegionResolver) ResolveRegion(col ImportColumn, displayValue string, parentRegionCode int64) (int64, error) {
	if code, ok := m.values[displayValue]; ok {
		return code, nil
	}
	return 0, ErrImportRefSkip
}

func TestParseImportRowsRegion(t *testing.T) {
	mapping := &ImportMapping{
		Version: 1,
		Columns: []ImportColumn{
			{Sort: 1, ExcelHeader: "省份", DbField: "province_code", FieldType: "region", Required: true, Importable: true, RegionLevel: 1},
		},
	}
	EnrichRegionColumns(mapping)
	region := &mockRegionResolver{values: map[string]int64{"江苏省": 320000}}
	valid, _, errs := ParseImportRows(mapping, []map[string]string{
		{"province_code": "江苏省"},
	}, []int{4}, nil, nil, region)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %+v", errs)
	}
	if len(valid) != 1 {
		t.Fatalf("want 1 valid row got %d", len(valid))
	}
	if valid[0]["province_code"] != int64(320000) {
		t.Fatalf("want province_code 320000 got %v", valid[0]["province_code"])
	}
}

func TestEnrichRegionColumnsInferLevelFromDbField(t *testing.T) {
	mapping := &ImportMapping{
		Columns: []ImportColumn{
			{DbField: "province_code", FieldType: "region"},
			{DbField: "city_code", FieldType: "region"},
			{DbField: "area_code", FieldType: "region"},
		},
	}
	EnrichRegionColumns(mapping)
	if EffectiveRegionLevel(mapping.Columns[0]) != 1 {
		t.Fatalf("province level want 1 got %d", EffectiveRegionLevel(mapping.Columns[0]))
	}
	if EffectiveRegionLevel(mapping.Columns[1]) != 2 {
		t.Fatalf("city level want 2 got %d", EffectiveRegionLevel(mapping.Columns[1]))
	}
	if EffectiveRegionLevel(mapping.Columns[2]) != 3 {
		t.Fatalf("area level want 3 got %d", EffectiveRegionLevel(mapping.Columns[2]))
	}
	if mapping.Columns[1].ParentField != "province_code" {
		t.Fatalf("city parent want province_code got %q", mapping.Columns[1].ParentField)
	}
}

func TestResolveRegionImportFieldTypeFromSearchtype(t *testing.T) {
	ft := resolveRegionImportFieldType(GenerateCodeField{
		Field: "city_code", Formtype: "region", Searchtype: "region_city",
	})
	if ft != "region_city" {
		t.Fatalf("want region_city got %s", ft)
	}
}
