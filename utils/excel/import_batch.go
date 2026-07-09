package excel

import "fmt"

// BatchExecuteStats 分批导入结果
type BatchExecuteStats struct {
	ImportExecuteStats
	CompletedBatches int
	TotalBatches     int
	FailedBatch      int // 失败批次号（0-based），-1 表示无失败
}

// BatchExecuteImportRows 按批执行导入；startBatch 为续导起始批次（0-based）
func BatchExecuteImportRows(
	mapping *ImportMapping,
	rows []map[string]interface{},
	opts ImportExecuteOpts,
	w ImportDBWriter,
	batchSize int,
	startBatch int,
) (BatchExecuteStats, error) {
	var out BatchExecuteStats
	if batchSize <= 0 {
		batchSize = 500
	}
	if startBatch < 0 {
		startBatch = 0
	}
	if len(rows) == 0 {
		return out, fmt.Errorf("无有效数据行")
	}
	total := (len(rows) + batchSize - 1) / batchSize
	out.TotalBatches = total

	bw, hasTx := w.(ImportBatchWriter)
	for b := startBatch; b < total; b++ {
		start := b * batchSize
		end := start + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		slice := rows[start:end]
		batchOpts := opts
		batchOpts.ExcelRowNums = sliceExcelRowNums(opts.ExcelRowNums, start, end)

		if hasTx {
			if err := bw.BeginBatch(); err != nil {
				out.FailedBatch = b
				return out, fmt.Errorf("批次 %d 开启事务失败: %w", b+1, err)
			}
		}
		stats, err := ExecuteImportRows(mapping, slice, batchOpts, w)
		mergeExecuteStats(&out.ImportExecuteStats, stats)
		if err != nil {
			if hasTx {
				_ = bw.RollbackBatch()
			}
			out.CompletedBatches = b
			out.FailedBatch = b
			return out, fmt.Errorf("批次 %d 失败: %w", b+1, err)
		}
		if hasTx {
			if err := bw.CommitBatch(); err != nil {
				out.FailedBatch = b
				return out, fmt.Errorf("批次 %d 提交失败: %w", b+1, err)
			}
		}
		out.CompletedBatches = b + 1
	}
	out.FailedBatch = -1
	return out, nil
}

// ImportBatchWriter 可选批次事务（由 DBImportExecutor 实现）
type ImportBatchWriter interface {
	ImportDBWriter
	BeginBatch() error
	CommitBatch() error
	RollbackBatch() error
}

func sliceExcelRowNums(nums []int, start, end int) []int {
	if len(nums) == 0 {
		return nil
	}
	if start >= len(nums) {
		return nil
	}
	if end > len(nums) {
		end = len(nums)
	}
	return nums[start:end]
}

func mergeExecuteStats(dst *ImportExecuteStats, src ImportExecuteStats) {
	dst.InsertRows += src.InsertRows
	dst.UpdateRows += src.UpdateRows
	dst.SkipRows += src.SkipRows
	if len(src.SkipDetails) > 0 {
		dst.SkipDetails = append(dst.SkipDetails, src.SkipDetails...)
	}
}
