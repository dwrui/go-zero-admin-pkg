package excel

import (
	"fmt"
	"strings"
)

const (
	// ImportTemplateHeaderRow Excel 第 1 行：表头
	ImportTemplateHeaderRow = 1
	// ImportTemplateMetaRow Excel 第 2 行：模板说明与字段填写提示（不解析为数据）
	ImportTemplateMetaRow = 2
	// ImportTemplateDataStartRow Excel 第 3 行起：填写数据
	ImportTemplateDataStartRow = 3
)

const ImportTemplateMetaPrefix = "模板:"

// ImportTemplateMetaLine 第 2 行模板说明文案
func ImportTemplateMetaLine(tplName string) string {
	tplName = strings.TrimSpace(tplName)
	if tplName == "" {
		return ImportTemplateMetaPrefix
	}
	return ImportTemplateMetaPrefix + tplName
}

// ImportTemplateDataStartIndex 数据区起始下标（0-based），固定为 Excel 第 3 行
func ImportTemplateDataStartIndex() int {
	return ImportTemplateDataStartRow - 1
}

// IsImportTemplateMetaRow 是否为模板标识行
func IsImportTemplateMetaRow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(cells[0]), ImportTemplateMetaPrefix)
}

// ValidateImportTemplateLayout 校验导入模板至少有表头与说明行，数据从第 3 行起
func ValidateImportTemplateLayout(rows [][]string) error {
	if len(rows) < ImportTemplateDataStartRow {
		return fmt.Errorf("请使用系统下载的导入模板：第1行表头、第2行填写说明、第3行起填数据")
	}
	return nil
}
