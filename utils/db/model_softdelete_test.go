package db

import (
	"context"
	"strings"
	"testing"
)

func testModelWithDeleteTime(table string) *Model {
	has := true
	return &Model{
		table:         table,
		fields:        []string{"*"},
		hasDeleteTime: &has,
	}
}

func TestBuildQuerySoftDeleteScope(t *testing.T) {
	ctx := context.Background()

	t.Run("default excludes trashed", func(t *testing.T) {
		m := testModelWithDeleteTime("ga_test")
		query, _ := m.buildQuery(ctx)
		if !strings.Contains(query, "delete_time = 0") {
			t.Fatalf("expected delete_time = 0, got: %s", query)
		}
	})

	t.Run("with trashed includes all", func(t *testing.T) {
		m := testModelWithDeleteTime("ga_test").WithTrashed()
		query, _ := m.buildQuery(ctx)
		if strings.Contains(query, "delete_time") {
			t.Fatalf("expected no delete_time filter, got: %s", query)
		}
	})

	t.Run("only trashed", func(t *testing.T) {
		m := testModelWithDeleteTime("ga_test").OnlyTrashed()
		query, _ := m.buildQuery(ctx)
		if !strings.Contains(query, "delete_time > 0") {
			t.Fatalf("expected delete_time > 0, got: %s", query)
		}
	})
}

func TestDeleteSoftDeleteSQL(t *testing.T) {
	ctx := context.Background()
	m := testModelWithDeleteTime("ga_test").Where("id", 1).SQLFetch(true)
	res := m.Delete(ctx)
	if res.GetError() != nil {
		t.Fatalf("unexpected err: %v", res.GetError())
	}
	query := res.GetSQL()
	if !strings.HasPrefix(query, "UPDATE ") {
		t.Fatalf("expected soft delete UPDATE, got: %s", query)
	}
	if !strings.Contains(query, softDeleteTimeExpr) {
		t.Fatalf("expected soft delete timestamp expr, got: %s", query)
	}
	if !strings.Contains(query, "delete_time = 0") {
		t.Fatalf("expected only update active rows, got: %s", query)
	}
}

func TestForceDeleteSQL(t *testing.T) {
	ctx := context.Background()
	m := testModelWithDeleteTime("ga_test").Where("id", 1).SQLFetch(true)
	res := m.ForceDelete(ctx)
	if res.GetError() != nil {
		t.Fatalf("unexpected err: %v", res.GetError())
	}
	query := res.GetSQL()
	if !strings.HasPrefix(query, "DELETE FROM ") {
		t.Fatalf("expected DELETE FROM, got: %s", query)
	}
}

func TestRestoreSQL(t *testing.T) {
	ctx := context.Background()
	m := testModelWithDeleteTime("ga_test").Where("id", 1).SQLFetch(true)
	res := m.Restore(ctx)
	if res.GetError() != nil {
		t.Fatalf("unexpected err: %v", res.GetError())
	}
	query := res.GetSQL()
	if !strings.Contains(query, "`delete_time` = ?") && !strings.Contains(query, "delete_time = ?") {
		t.Fatalf("expected delete_time reset, got: %s", query)
	}
	if !strings.Contains(query, "delete_time > 0") {
		t.Fatalf("expected only trashed rows, got: %s", query)
	}
}

func TestRestoreRequiresWhere(t *testing.T) {
	ctx := context.Background()
	m := testModelWithDeleteTime("ga_test")
	res := m.Restore(ctx)
	if res.GetError() == nil {
		t.Fatal("expected error when restore without where")
	}
}

func TestUpdateSoftDeleteScope(t *testing.T) {
	ctx := context.Background()
	m := testModelWithDeleteTime("ga_test").Where("id", 1).SQLFetch(true)
	res := m.Update(ctx, map[string]interface{}{"status": 1})
	if res.GetError() != nil {
		t.Fatalf("unexpected err: %v", res.GetError())
	}
	if !strings.Contains(res.GetSQL(), "delete_time = 0") {
		t.Fatalf("expected delete_time = 0 on update, got: %s", res.GetSQL())
	}
}
