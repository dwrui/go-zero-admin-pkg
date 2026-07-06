package excel

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ImportRowError 单行字段解析错误（Excel 行号为 1-based）
type ImportRowError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ParseImportRows 按 mapping 校验并转换原始行
func ParseImportRows(mapping *ImportMapping, rawRows []map[string]string, excelRowNums []int, belongTo BelongToResolver, dic DicResolver) ([]map[string]interface{}, []ImportRowError) {
	if mapping == nil {
		return nil, []ImportRowError{{Row: 0, Field: "", Message: "mapping 为空"}}
	}
	cols := mapping.ImportableColumns()
	colByField := make(map[string]ImportColumn, len(cols))
	for _, c := range cols {
		colByField[c.DbField] = c
	}

	valid := make([]map[string]interface{}, 0, len(rawRows))
	var allErrors []ImportRowError

	for i, raw := range rawRows {
		excelRow := i + 2
		if i < len(excelRowNums) && excelRowNums[i] > 0 {
			excelRow = excelRowNums[i]
		}
		converted, rowErrors := parseOneImportRow(colByField, cols, excelRow, raw, belongTo, dic)
		if len(rowErrors) > 0 {
			allErrors = append(allErrors, rowErrors...)
			continue
		}
		if len(converted) > 0 {
			valid = append(valid, converted)
		}
	}
	return valid, allErrors
}

func parseOneImportRow(colByField map[string]ImportColumn, cols []ImportColumn, excelRow int, raw map[string]string, belongTo BelongToResolver, dic DicResolver) (map[string]interface{}, []ImportRowError) {
	out := make(map[string]interface{})
	var errs []ImportRowError

	for _, col := range cols {
		rawVal := strings.TrimSpace(raw[col.DbField])
		if rawVal == "" && !col.Required {
			continue
		}
		converted, err := ConvertImportCell(col, rawVal, belongTo, dic)
		if err == ErrImportRefSkip {
			continue
		}
		if err != nil {
			errs = append(errs, ImportRowError{
				Row:     excelRow,
				Field:   col.DbField,
				Message: err.Error(),
			})
			continue
		}
		if converted == nil {
			continue
		}
		out[col.DbField] = converted
	}

	// 必填列未出现在 raw 中
	for _, col := range cols {
		if !col.Required {
			continue
		}
		if _, ok := out[col.DbField]; ok {
			continue
		}
		if strings.TrimSpace(raw[col.DbField]) != "" {
			continue
		}
		errs = append(errs, ImportRowError{
			Row:     excelRow,
			Field:   col.DbField,
			Message: fmt.Sprintf("%s 为必填项", col.ExcelHeader),
		})
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return out, nil
}

// ConvertImportCell 将单元格字符串转为入库值
func ConvertImportCell(col ImportColumn, raw string, belongTo BelongToResolver, dic DicResolver) (interface{}, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if col.Required {
			return nil, fmt.Errorf("%s 为必填项", col.ExcelHeader)
		}
		return nil, nil
	}

	switch col.FieldType {
	case "number":
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s 须为整数", col.ExcelHeader)
		}
		return n, nil
	case "float":
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("%s 须为数字", col.ExcelHeader)
		}
		return f, nil
	case "radio", "select", "switch":
		return resolveOptionValue(col.OptionValue, raw, col.ExcelHeader)
	case "belongDic":
		if col.DicGroupID > 0 {
			if dic == nil {
				return nil, fmt.Errorf("%s 需要字典解析器", col.ExcelHeader)
			}
			return dic.ResolveBelongDic(col, raw)
		}
		// 未配置 dic_group_id 时回退 option_value（兼容旧字段）
		return resolveOptionValue(col.OptionValue, raw, col.ExcelHeader)
	case "belongto":
		if belongTo == nil {
			return nil, fmt.Errorf("%s 需要关联解析器", col.ExcelHeader)
		}
		id, err := belongTo.ResolveBelongTo(col, raw)
		if err != nil {
			return nil, err
		}
		return id, nil
	case "date":
		return normalizeDate(raw, col.ExcelHeader)
	case "datetime":
		return normalizeDateTime(raw, col.ExcelHeader)
	case "time":
		return normalizeTime(raw, col.ExcelHeader)
	default:
		return raw, nil
	}
}

func resolveOptionValue(optionValue, input, header string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", nil
	}
	if strings.TrimSpace(optionValue) == "" {
		return input, nil
	}
	for _, part := range strings.Split(optionValue, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		label := strings.TrimSpace(kv[1])
		if input == key || input == label {
			return key, nil
		}
	}
	return "", fmt.Errorf("%s 无效选项「%s」，可选: %s", header, input, optionValue)
}

var dateLayouts = []string{
	"2006-01-02",
	"2006/01/02",
	"2006-1-2",
	"2006/1/2",
}

var dateTimeLayouts = []string{
	"2006-01-02 15:04:05",
	"2006/01/02 15:04:05",
	"2006-01-02 15:04",
	"2006/01/02 15:04",
	"2006-01-02",
	"2006/01/02",
}

var timeLayouts = []string{
	"15:04:05",
	"15:04",
}

func normalizeDate(raw, header string) (string, error) {
	t, err := parseFlexibleTime(raw, dateLayouts)
	if err != nil {
		return "", fmt.Errorf("%s 日期格式无效，请使用 YYYY-MM-DD", header)
	}
	return t.Format("2006-01-02"), nil
}

func normalizeDateTime(raw, header string) (string, error) {
	t, err := parseFlexibleTime(raw, dateTimeLayouts)
	if err != nil {
		return "", fmt.Errorf("%s 日期时间格式无效", header)
	}
	if !strings.Contains(raw, ":") {
		return t.Format("2006-01-02") + " 00:00:00", nil
	}
	return t.Format("2006-01-02 15:04:05"), nil
}

func normalizeTime(raw, header string) (string, error) {
	t, err := parseFlexibleTime(raw, timeLayouts)
	if err != nil {
		return "", fmt.Errorf("%s 时间格式无效，请使用 HH:mm:ss", header)
	}
	return t.Format("15:04:05"), nil
}

func parseFlexibleTime(raw string, layouts []string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	var lastErr error
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, raw, time.Local)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return time.Time{}, lastErr
	}
	return time.Time{}, fmt.Errorf("无法解析时间")
}
