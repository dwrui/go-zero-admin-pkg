// Package datacenter 导入导出平台能力（先在 common 内实现，后续整体迁移至 datacenter.rpc）。
package datacenter

const (
	// ConfigCategoryKey 配置中心分类键
	ConfigCategoryKey = "datacenter_import_export"
)

// Settings 从配置中心读取的导入导出参数（均有默认值，配置缺失时不写死业务逻辑）
type Settings struct {
	SyncExportMaxRows       int
	SyncImportMaxRows       int
	MaxFileBytes            int64
	ImportBatchSize         int
	ImportKeyTTLSec         int
	GlobalMaxRunning        int
	TenantMaxRunning        int
	UserDailyMaxTasks       int
	ApprovalExportMinRows   int
	ApprovalSensitiveExport bool
	ImportDiffEnabled   bool
	ImportDiffBizTypes  string
	ResultExpireDays    int
	OssRetainDays           int
	OssEnabled              bool
	OssStorageName          string
	ExportFilenamePattern   string
}

// DefaultSettings 与 goza_erp_add.sql 种子数据一致
func DefaultSettings() Settings {
	return Settings{
		SyncExportMaxRows:       5000,
		SyncImportMaxRows:       2000,
		MaxFileBytes:            10 * 1024 * 1024,
		ImportBatchSize:         500,
		ImportKeyTTLSec:         86400,
		GlobalMaxRunning:        10,
		TenantMaxRunning:        3,
		UserDailyMaxTasks:       50,
		ApprovalExportMinRows:   10000,
		ApprovalSensitiveExport: true,
		ImportDiffEnabled:       false,
		ImportDiffBizTypes:      "customer,supplier,material",
		ResultExpireDays:        7,
		OssRetainDays:           90,
		OssEnabled:              false,
		OssStorageName:          "",
		ExportFilenamePattern:   DefaultExportFilenamePattern,
	}
}
