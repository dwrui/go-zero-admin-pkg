package datacenter

import (
	"strconv"
	"strings"
)

func applyInt(m map[string]string, key string, fallback int) int {
	if m == nil {
		return fallback
	}
	raw := strings.TrimSpace(m[key])
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func applyInt64(m map[string]string, key string, fallback int64) int64 {
	if m == nil {
		return fallback
	}
	raw := strings.TrimSpace(m[key])
	if raw == "" {
		return fallback
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func applyBool(m map[string]string, key string, fallback bool) bool {
	if m == nil {
		return fallback
	}
	raw := strings.TrimSpace(strings.ToLower(m[key]))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

// SettingsFromMap 将配置中心键值对转为 Settings
func SettingsFromMap(m map[string]string) Settings {
	def := DefaultSettings()
	return Settings{
		SyncExportMaxRows:       applyInt(m, "sync_export_max_rows", def.SyncExportMaxRows),
		SyncImportMaxRows:       applyInt(m, "sync_import_max_rows", def.SyncImportMaxRows),
		MaxFileBytes:            applyInt64(m, "max_file_bytes", def.MaxFileBytes),
		ImportBatchSize:         applyInt(m, "import_batch_size", def.ImportBatchSize),
		ImportKeyTTLSec:         applyInt(m, "import_key_ttl_sec", def.ImportKeyTTLSec),
		GlobalMaxRunning:        applyInt(m, "global_max_running", def.GlobalMaxRunning),
		TenantMaxRunning:        applyInt(m, "tenant_max_running", def.TenantMaxRunning),
		UserDailyMaxTasks:       applyInt(m, "user_daily_max_tasks", def.UserDailyMaxTasks),
		ApprovalExportMinRows:   applyInt(m, "approval_export_min_rows", def.ApprovalExportMinRows),
		ApprovalSensitiveExport: applyBool(m, "approval_sensitive_export", def.ApprovalSensitiveExport),
		ImportDiffEnabled:       applyBool(m, "import_diff_enabled", def.ImportDiffEnabled),
		ImportDiffBizTypes:      applyString(m, "import_diff_biz_types", def.ImportDiffBizTypes),
		ResultExpireDays:        applyInt(m, "result_expire_days", def.ResultExpireDays),
		OssRetainDays:           applyInt(m, "oss_retain_days", def.OssRetainDays),
		OssEnabled:              applyBool(m, "oss_enabled", def.OssEnabled),
		OssStorageName:          strings.TrimSpace(m["oss_storage_name"]),
		ExportFilenamePattern:   applyString(m, "export_filename_pattern", def.ExportFilenamePattern),
	}
}

func applyString(m map[string]string, key, fallback string) string {
	if m == nil {
		return fallback
	}
	raw := strings.TrimSpace(m[key])
	if raw == "" {
		return fallback
	}
	return raw
}

// ShouldAsyncExport 是否应走异步导出
func (s Settings) ShouldAsyncExport(rowCount int) bool {
	if rowCount <= 0 {
		return false
	}
	return rowCount > s.SyncExportMaxRows
}

// ShouldAsyncImport 是否应走异步导入
func (s Settings) ShouldAsyncImport(rowCount int) bool {
	if rowCount <= 0 {
		return false
	}
	return rowCount > s.SyncImportMaxRows
}

// ShouldApproveExport 是否需审批（行数阈值；敏感模板由调用方叠加判断）
func (s Settings) ShouldApproveExport(rowCount int) bool {
	if s.ApprovalExportMinRows <= 0 {
		return false
	}
	return rowCount >= s.ApprovalExportMinRows
}
