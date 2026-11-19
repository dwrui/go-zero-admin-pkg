package datacenter

const (
	// ExportQueueKey Redis 异步导出任务待处理队列
	ExportQueueKey = "datacenter:export:queue"
	// ExportProcessingKey Redis 异步导出处理中队列（RPOPLPUSH/ACK）
	ExportProcessingKey = "datacenter:export:processing"
	// ExportResultKeyPrefix 导出结果临时存储（S2 未接 OSS 前）
	ExportResultKeyPrefix = "datacenter:export:result:"
	// ImportQueueKey Redis 异步导入任务待处理队列
	ImportQueueKey = "datacenter:import:queue"
	// ImportProcessingKey Redis 异步导入处理中队列
	ImportProcessingKey = "datacenter:import:processing"
	// ImportErrorKeyPrefix 导入错误明细 Redis key 前缀
	ImportErrorKeyPrefix = "datacenter:import:error:"
	// CleanupLockKey 过期文件清理分布式锁（多实例仅一个执行）
	CleanupLockKey = "datacenter:cleanup:lock"
)

// ReclaimLockKey 启动时 processing 回收分布式锁（按队列区分，多实例仅一个执行 startup reclaim）。
func ReclaimLockKey(queueKey string) string {
	return "datacenter:reclaim:lock:" + queueKey
}

// ExportResultTTL 导出结果 Redis 保留秒数（与 result_expire_days 对齐，默认 7 天）
const ExportResultTTL = 7 * 86400
