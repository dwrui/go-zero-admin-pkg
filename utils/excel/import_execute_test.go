package excel

import "testing"

type mockImportDBWriter struct {
	existing map[string]int64
	columns  map[string]struct{}
	inserted []map[string]interface{}
	updated  []map[string]interface{}
}

func (m *mockImportDBWriter) key(table string, keys []string, row map[string]interface{}) string {
	k := table
	for _, f := range keys {
		k += "|" + f + "=" + row[f].(string)
	}
	return k
}

func (m *mockImportDBWriter) FindByDuplicateKey(table string, businessId int64, keys []string, row map[string]interface{}) (int64, bool, error) {
	if !DuplicateKeyComplete(keys, row) {
		return 0, false, nil
	}
	id, ok := m.existing[m.key(table, keys, row)]
	return id, ok, nil
}

func (m *mockImportDBWriter) InsertRow(table string, data map[string]interface{}) error {
	m.inserted = append(m.inserted, data)
	return nil
}

func (m *mockImportDBWriter) UpdateRow(table string, id int64, data map[string]interface{}) error {
	cp := map[string]interface{}{"id": id}
	for k, v := range data {
		cp[k] = v
	}
	m.updated = append(m.updated, cp)
	return nil
}

func (m *mockImportDBWriter) TableHasColumn(table, column string) bool {
	_, ok := m.columns[column]
	return ok
}

func TestExecuteImportRowsUpsertUpdate(t *testing.T) {
	mapping := &ImportMapping{
		TargetTable:  "ga_goods_brand",
		ImportMode:   "upsert",
		DuplicateKey: []string{"brand_code"},
	}
	writer := &mockImportDBWriter{
		existing: map[string]int64{"ga_goods_brand|brand_code=OLD": 10},
		columns: map[string]struct{}{
			"business_id": {}, "dept_id": {}, "create_by": {}, "update_by": {}, "brand_code": {}, "brand_name": {},
		},
	}
	rows := []map[string]interface{}{
		{"brand_code": "NEW", "brand_name": "新品牌"},
		{"brand_code": "OLD", "brand_name": "旧品牌改"},
	}
	stats, err := ExecuteImportRows(mapping, rows, ImportExecuteOpts{
		BusinessId: 1, UserId: 100, DeptId: 5, DuplicateMode: "update",
	}, writer)
	if err != nil {
		t.Fatal(err)
	}
	if stats.InsertRows != 1 || stats.UpdateRows != 1 || stats.SkipRows != 0 {
		t.Fatalf("stats: %+v", stats)
	}
	if writer.inserted[0]["create_by"] != int64(100) {
		t.Fatalf("create_by not injected: %+v", writer.inserted[0])
	}
	if writer.updated[0]["id"] != int64(10) {
		t.Fatalf("update wrong id: %+v", writer.updated[0])
	}
}

func TestExecuteImportRowsUpsertSkip(t *testing.T) {
	mapping := &ImportMapping{
		TargetTable:  "ga_goods_brand",
		ImportMode:   "upsert",
		DuplicateKey: []string{"brand_code"},
	}
	writer := &mockImportDBWriter{
		existing: map[string]int64{"ga_goods_brand|brand_code=DUP": 1},
		columns:  map[string]struct{}{"business_id": {}, "brand_code": {}},
	}
	stats, err := ExecuteImportRows(mapping, []map[string]interface{}{
		{"brand_code": "DUP", "brand_name": "x"},
	}, ImportExecuteOpts{BusinessId: 1, DuplicateMode: "skip"}, writer)
	if err != nil {
		t.Fatal(err)
	}
	if stats.SkipRows != 1 || stats.InsertRows != 0 {
		t.Fatalf("stats: %+v", stats)
	}
}

func TestExecuteImportRowsUpdateOnly(t *testing.T) {
	mapping := &ImportMapping{
		TargetTable:  "ga_goods_brand",
		ImportMode:   "update",
		DuplicateKey: []string{"brand_code"},
	}
	writer := &mockImportDBWriter{
		existing: map[string]int64{},
		columns:  map[string]struct{}{"business_id": {}, "brand_code": {}, "update_by": {}},
	}
	stats, err := ExecuteImportRows(mapping, []map[string]interface{}{
		{"brand_code": "MISS", "brand_name": "x"},
	}, ImportExecuteOpts{BusinessId: 1, UserId: 2, DuplicateMode: "update"}, writer)
	if err != nil {
		t.Fatal(err)
	}
	if stats.SkipRows != 1 {
		t.Fatalf("want skip 1 got %+v", stats)
	}
}
