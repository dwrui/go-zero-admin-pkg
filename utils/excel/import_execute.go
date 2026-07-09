package excel

import (
	"fmt"
	"strings"
)

// ImportExecuteOpts 导入执行上下文
type ImportExecuteOpts struct {
	BusinessId    int64
	UserId        int64
	DeptId        int64
	DuplicateMode string // skip | stop | update
	ExcelRowNums  []int  // 与 rows 对齐的 Excel 行号（可选）
	TaskId        uint64 // 导入任务 ID（>0 时写入变更快照）
}

// ImportRowUpdateMeta 更新行上下文（变更快照）
type ImportRowUpdateMeta struct {
	ExcelRow int32
	TaskId   uint64
}

// ImportContextWriter 支持带行号的更新（可选，由 common DBImportExecutor 实现）
type ImportContextWriter interface {
	ImportDBWriter
	UpdateRowWithContext(table string, id int64, data map[string]interface{}, meta ImportRowUpdateMeta) error
}

// ImportSkipDetail 跳过/失败行明细
type ImportSkipDetail struct {
	Row     int32
	Message string
}

// ImportExecuteStats 导入结果统计
type ImportExecuteStats struct {
	InsertRows  int32
	UpdateRows  int32
	SkipRows    int32
	SkipDetails []ImportSkipDetail
}

// ImportDBWriter 导入写库（由 common 服务实现）
type ImportDBWriter interface {
	FindByDuplicateKey(table string, businessId int64, keys []string, row map[string]interface{}) (int64, bool, error)
	InsertRow(table string, data map[string]interface{}) error
	UpdateRow(table string, id int64, data map[string]interface{}) error
	TableHasColumn(table, column string) bool
}

// ExecuteImportRows 按 mapping 与重复策略写入数据
func ExecuteImportRows(mapping *ImportMapping, rows []map[string]interface{}, opts ImportExecuteOpts, w ImportDBWriter) (ImportExecuteStats, error) {
	var stats ImportExecuteStats
	if mapping == nil || w == nil {
		return stats, fmt.Errorf("mapping 或 writer 为空")
	}
	if len(rows) == 0 {
		return stats, fmt.Errorf("无有效数据行")
	}
	dupMode := normalizeDuplicateMode(opts.DuplicateMode)
	importMode := normalizeImportMode(mapping.ImportMode)
	table := mapping.TargetTable
	dupKeys := mapping.DuplicateKey

	for i, row := range rows {
		rowNo := i + 1
		excelRow := int32(rowNo)
		if i < len(opts.ExcelRowNums) && opts.ExcelRowNums[i] > 0 {
			excelRow = int32(opts.ExcelRowNums[i])
		}
		action, existingID, err := resolveImportAction(importMode, dupMode, dupKeys, table, opts.BusinessId, row, w)
		if err != nil {
			return stats, fmt.Errorf("第 %d 行: %w", excelRow, err)
		}
		switch action {
		case importActionSkip:
			stats.SkipRows++
			stats.SkipDetails = append(stats.SkipDetails, ImportSkipDetail{
				Row:     excelRow,
				Message: "重复数据，已跳过",
			})
		case importActionInsert:
			data := prepareImportRowData(row, opts, w, table, false)
			if err := w.InsertRow(table, data); err != nil {
				handled, handleErr := handleInsertConflict(table, dupKeys, dupMode, int(excelRow), row, opts, w, &stats)
				if handleErr != nil {
					return stats, handleErr
				}
				if handled {
					continue
				}
				if dupMode == "stop" {
					return stats, fmt.Errorf("第 %d 行写入失败: %v", excelRow, err)
				}
				stats.SkipRows++
				stats.SkipDetails = append(stats.SkipDetails, ImportSkipDetail{
					Row:     excelRow,
					Message: fmt.Sprintf("写入失败: %v", err),
				})
				continue
			}
			stats.InsertRows++
		case importActionUpdate:
			data := prepareImportRowData(row, opts, w, table, true)
			if err := updateImportRow(w, table, existingID, data, ImportRowUpdateMeta{
				ExcelRow: excelRow,
				TaskId:   opts.TaskId,
			}); err != nil {
				if dupMode == "stop" {
					return stats, fmt.Errorf("第 %d 行更新失败: %v", excelRow, err)
				}
				stats.SkipRows++
				stats.SkipDetails = append(stats.SkipDetails, ImportSkipDetail{
					Row:     excelRow,
					Message: fmt.Sprintf("更新失败: %v", err),
				})
				continue
			}
			stats.UpdateRows++
		}
	}
	return stats, nil
}

type importAction string

const (
	importActionInsert importAction = "insert"
	importActionUpdate importAction = "update"
	importActionSkip   importAction = "skip"
)

func normalizeDuplicateMode(mode string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	switch mode {
	case "stop", "update":
		return mode
	default:
		return "skip"
	}
}

func normalizeImportMode(mode string) string {
	mode = strings.TrimSpace(strings.ToLower(mode))
	switch mode {
	case "insert", "update":
		return mode
	default:
		return "upsert"
	}
}

func resolveImportAction(importMode, dupMode string, dupKeys []string, table string, businessId int64, row map[string]interface{}, w ImportDBWriter) (importAction, int64, error) {
	if importMode == "insert" || len(dupKeys) == 0 {
		return importActionInsert, 0, nil
	}
	existingID, found, err := w.FindByDuplicateKey(table, businessId, dupKeys, row)
	if err != nil {
		return "", 0, err
	}
	switch importMode {
	case "update":
		if !found {
			return importActionSkip, 0, nil
		}
		if dupMode == "stop" {
			return "", 0, fmt.Errorf("记录不存在，无法更新")
		}
		if dupMode == "skip" {
			return importActionSkip, 0, nil
		}
		return importActionUpdate, existingID, nil
	default: // upsert
		if !found {
			return importActionInsert, 0, nil
		}
		switch dupMode {
		case "stop":
			return "", 0, fmt.Errorf("重复数据")
		case "skip":
			return importActionSkip, 0, nil
		case "update":
			return importActionUpdate, existingID, nil
		default:
			return importActionSkip, 0, nil
		}
	}
}

func handleInsertConflict(table string, dupKeys []string, dupMode string, rowNo int, row map[string]interface{}, opts ImportExecuteOpts, w ImportDBWriter, stats *ImportExecuteStats) (bool, error) {
	if dupMode != "update" || len(dupKeys) == 0 {
		return false, nil
	}
	existingID, found, err := w.FindByDuplicateKey(table, opts.BusinessId, dupKeys, row)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	data := prepareImportRowData(row, opts, w, table, true)
	excelRow := int32(rowNo)
	if err := updateImportRow(w, table, existingID, data, ImportRowUpdateMeta{
		ExcelRow: excelRow,
		TaskId:   opts.TaskId,
	}); err != nil {
		if dupMode == "stop" {
			return false, fmt.Errorf("第 %d 行更新失败: %v", rowNo, err)
		}
		stats.SkipRows++
		return true, nil
	}
	stats.UpdateRows++
	return true, nil
}

func updateImportRow(w ImportDBWriter, table string, id int64, data map[string]interface{}, meta ImportRowUpdateMeta) error {
	if cw, ok := w.(ImportContextWriter); ok && meta.TaskId > 0 {
		return cw.UpdateRowWithContext(table, id, data, meta)
	}
	return w.UpdateRow(table, id, data)
}

func prepareImportRowData(row map[string]interface{}, opts ImportExecuteOpts, w ImportDBWriter, table string, forUpdate bool) map[string]interface{} {
	skip := map[string]struct{}{
		"id": {}, "create_time": {}, "update_time": {}, "delete_time": {},
	}
	if forUpdate {
		skip["business_id"] = struct{}{}
		skip["create_by"] = struct{}{}
		skip["dept_id"] = struct{}{}
	}
	data := make(map[string]interface{}, len(row)+4)
	for k, v := range row {
		if _, ok := skip[k]; ok {
			continue
		}
		data[k] = v
	}
	if !forUpdate {
		if w.TableHasColumn(table, "business_id") {
			data["business_id"] = opts.BusinessId
		}
		if opts.DeptId > 0 && w.TableHasColumn(table, "dept_id") {
			data["dept_id"] = opts.DeptId
		}
		if opts.UserId > 0 {
			if w.TableHasColumn(table, "create_by") {
				data["create_by"] = opts.UserId
			}
			if w.TableHasColumn(table, "update_by") {
				data["update_by"] = opts.UserId
			}
		}
	} else if opts.UserId > 0 && w.TableHasColumn(table, "update_by") {
		data["update_by"] = opts.UserId
	}
	return data
}

// DuplicateKeyComplete 重复键字段是否齐全且非空
func DuplicateKeyComplete(keys []string, row map[string]interface{}) bool {
	if len(keys) == 0 {
		return false
	}
	for _, k := range keys {
		v, ok := row[k]
		if !ok || v == nil {
			return false
		}
		if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
			return false
		}
	}
	return true
}
