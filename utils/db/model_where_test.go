package db

import (
	"strings"
	"testing"
)

func renderWhereForTest(qb *Model) (string, []interface{}) {
	var parts []string
	var args []interface{}
	for i, where := range qb.where {
		if i > 0 {
			parts = append(parts, " "+where.operator+" ")
		}
		if where.field == "" {
			parts = append(parts, where.cond)
		} else {
			parts = append(parts, quoteField(where.field)+" "+where.cond)
		}
		args = append(args, where.args...)
	}
	return strings.Join(parts, ""), args
}

func TestWhereRawSQLWithoutArgs(t *testing.T) {
	m := &Model{}
	m.Where("tablename <> '' AND api_filename <> ''")
	if m.buildErr != nil {
		t.Fatalf("unexpected buildErr: %v", m.buildErr)
	}
	if len(m.where) != 1 {
		t.Fatalf("expected 1 where clause, got %d", len(m.where))
	}
	w := m.where[0]
	if w.field != "" {
		t.Fatalf("expected empty field, got %q", w.field)
	}
	if w.cond != "tablename <> '' AND api_filename <> ''" {
		t.Fatalf("unexpected cond: %q", w.cond)
	}

	query, args := renderWhereForTest(m)
	if query != "tablename <> '' AND api_filename <> ''" {
		t.Fatalf("unexpected where sql: %q", query)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %v", args)
	}
}

func TestWherePlaceholderWithoutArgs(t *testing.T) {
	m := &Model{}
	m.Where("status = ?")
	if m.buildErr == nil {
		t.Fatal("expected buildErr for placeholder without args")
	}
}

func TestWhereFieldEquals(t *testing.T) {
	m := &Model{}
	m.Where("status", 1)
	query, args := renderWhereForTest(m)
	if query != "`status` = ?" {
		t.Fatalf("unexpected query: %q", query)
	}
	if len(args) != 1 || args[0] != 1 {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestWherePlaceholderWithArgs(t *testing.T) {
	m := &Model{}
	m.Where("status = ? AND dept_id = ?", 1, 2)
	query, args := renderWhereForTest(m)
	if query != "status = ? AND dept_id = ?" {
		t.Fatalf("unexpected query: %q", query)
	}
	if len(args) != 2 || args[0] != 1 || args[1] != 2 {
		t.Fatalf("unexpected args: %v", args)
	}
}
