package db

import (
	"testing"
)

func TestCloneDoesNotShareWhere(t *testing.T) {
	base := &Model{
		table:  "ga_test",
		fields: []string{"*"},
	}
	base.Where("id", 1)

	clone := base.Clone()
	clone.Where("status", 1)

	if len(base.where) != 1 {
		t.Fatalf("expected base to keep 1 where clause, got %d", len(base.where))
	}
	if len(clone.where) != 2 {
		t.Fatalf("expected clone to have 2 where clauses, got %d", len(clone.where))
	}
}

func TestSortedMapKeys(t *testing.T) {
	keys := sortedMapKeys(map[string]interface{}{
		"c": 3,
		"a": 1,
		"b": 2,
	})
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Fatalf("unexpected key order: %v", keys)
	}
}
