package datacenter

import "testing"

func TestSettingsFromMap(t *testing.T) {
	s := SettingsFromMap(map[string]string{
		"sync_export_max_rows":    "8000",
		"approval_export_min_rows": "20000",
		"approval_sensitive_export": "0",
	})
	if s.SyncExportMaxRows != 8000 {
		t.Fatalf("SyncExportMaxRows=%d", s.SyncExportMaxRows)
	}
	if s.ApprovalExportMinRows != 20000 {
		t.Fatalf("ApprovalExportMinRows=%d", s.ApprovalExportMinRows)
	}
	if s.ApprovalSensitiveExport {
		t.Fatal("expected approval_sensitive_export false")
	}
	if !s.ShouldAsyncExport(8001) {
		t.Fatal("expected async export")
	}
	if s.ShouldAsyncExport(8000) {
		t.Fatal("expected sync at threshold")
	}
	if !s.ShouldApproveExport(20000) {
		t.Fatal("expected approve at threshold")
	}
}
