package datacenter

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExportWatermarkMeta 导出 Excel 水印元数据
type ExportWatermarkMeta struct {
	UserID   uint64
	UserName string
	BizType  string
	TaskNo   string
}

const exportMetaSheetName = "_export_meta"

// ApplyExportWatermark 为导出 xlsx 添加页眉/背景/隐藏元数据水印（失败时返回原始数据）
func ApplyExportWatermark(data []byte, meta ExportWatermarkMeta) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return data, nil
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return data, nil
	}
	mainSheet := sheets[0]

	userLabel := meta.UserName
	if userLabel == "" && meta.UserID > 0 {
		userLabel = fmt.Sprintf("用户%d", meta.UserID)
	}
	if userLabel == "" {
		userLabel = "未知"
	}
	exportTime := time.Now().Format("2006-01-02 15:04:05")
	header := fmt.Sprintf("导出人: %s  导出时间: %s", userLabel, exportTime)
	if meta.BizType != "" {
		header += fmt.Sprintf("  业务: %s", meta.BizType)
	}
	if meta.TaskNo != "" {
		header += fmt.Sprintf("  任务: %s", meta.TaskNo)
	}
	headerLine := "&C" + header
	_ = f.SetHeaderFooter(mainSheet, &excelize.HeaderFooterOptions{
		OddHeader: headerLine,
		OddFooter: "&C内部数据 请勿外传",
	})

	bgText := buildBackgroundWatermarkText(userLabel, exportTime, meta)
	for _, sheet := range sheets {
		if sheet == exportMetaSheetName || strings.HasPrefix(sheet, "_") {
			continue
		}
		applySheetBackgroundWatermark(f, sheet, bgText)
	}

	if err := ensureHiddenMetaSheet(f, meta, userLabel, exportTime); err != nil {
		return data, err
	}

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return data, err
	}
	return buf.Bytes(), nil
}

func ensureHiddenMetaSheet(f *excelize.File, meta ExportWatermarkMeta, userLabel, exportTime string) error {
	idx, err := f.GetSheetIndex(exportMetaSheetName)
	if err != nil || idx < 0 {
		if _, err := f.NewSheet(exportMetaSheetName); err != nil {
			return err
		}
	}
	_ = f.SetSheetVisible(exportMetaSheetName, false)
	rows := [][2]interface{}{
		{"导出操作人", userLabel},
		{"用户ID", meta.UserID},
		{"导出时间", exportTime},
		{"业务类型", meta.BizType},
		{"任务编号", meta.TaskNo},
	}
	for i, row := range rows {
		r := i + 1
		_ = f.SetCellValue(exportMetaSheetName, fmt.Sprintf("A%d", r), row[0])
		_ = f.SetCellValue(exportMetaSheetName, fmt.Sprintf("B%d", r), row[1])
	}
	return nil
}

func buildBackgroundWatermarkText(userLabel, exportTime string, meta ExportWatermarkMeta) string {
	line1 := fmt.Sprintf("%s  %s", userLabel, exportTime)
	if meta.BizType != "" {
		line1 += "  " + meta.BizType
	}
	line2 := "内部数据 请勿外传"
	if meta.TaskNo != "" {
		line2 += "  " + meta.TaskNo
	}
	return line1 + "\n" + line2
}
