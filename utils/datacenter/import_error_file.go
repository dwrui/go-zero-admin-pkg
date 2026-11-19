package datacenter

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// ImportErrorDetail 导入错误明细行
type ImportErrorDetail struct {
	Row     int
	Field   string
	Message string
}

// BuildImportErrorFile 生成导入错误明细 xlsx
func BuildImportErrorFile(summary string, details []ImportErrorDetail) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	if summary != "" {
		_ = f.SetSheetName("Sheet1", "摘要")
		_ = f.SetCellValue("摘要", "A1", "错误摘要")
		_ = f.SetCellValue("摘要", "A2", summary)
		_, _ = f.NewSheet("错误明细")
	} else {
		_ = f.SetSheetName("Sheet1", "错误明细")
	}

	sheet := "错误明细"
	_ = f.SetCellValue(sheet, "A1", "行号")
	_ = f.SetCellValue(sheet, "B1", "字段")
	_ = f.SetCellValue(sheet, "C1", "说明")
	row := 2
	for _, d := range details {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), d.Row)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), d.Field)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), d.Message)
		row++
	}
	if row == 2 && summary != "" {
		_ = f.SetCellValue(sheet, "A2", 0)
		_ = f.SetCellValue(sheet, "C2", summary)
	}

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
