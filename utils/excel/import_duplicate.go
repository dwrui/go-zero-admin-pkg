package excel

import (
	"fmt"
	"sort"
	"strings"
)

// CheckDuplicateKeyInFile 模板 duplicate_key 文件内跨行唯一性预检
func CheckDuplicateKeyInFile(mapping *ImportMapping, rawRows []map[string]string, excelRowNums []int) []ImportRowError {
	if mapping == nil || len(mapping.DuplicateKey) == 0 || len(rawRows) == 0 {
		return nil
	}
	type dupGroup struct {
		key  string
		rows []int
	}
	groups := make(map[string]*dupGroup)
	for i, raw := range rawRows {
		excelRow := i + 2
		if i < len(excelRowNums) && excelRowNums[i] > 0 {
			excelRow = excelRowNums[i]
		}
		composite := buildFileDuplicateKey(mapping.DuplicateKey, raw)
		if composite == "" {
			continue
		}
		g, ok := groups[composite]
		if !ok {
			g = &dupGroup{key: composite}
			groups[composite] = g
		}
		g.rows = append(g.rows, excelRow)
	}

	var errs []ImportRowError
	fieldLabel := strings.Join(mapping.DuplicateKey, "+")
	for _, g := range groups {
		if len(g.rows) < 2 {
			continue
		}
		sort.Ints(g.rows)
		for _, row := range g.rows {
			errs = append(errs, ImportRowError{
				Row:     row,
				Field:   fieldLabel,
				Message: fmt.Sprintf("文件内重复键「%s」与第 %s 行重复", g.key, formatOtherRows(g.rows, row)),
			})
		}
	}
	sort.Slice(errs, func(i, j int) bool {
		if errs[i].Row == errs[j].Row {
			return errs[i].Field < errs[j].Field
		}
		return errs[i].Row < errs[j].Row
	})
	return errs
}

func buildFileDuplicateKey(keys []string, raw map[string]string) string {
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := strings.TrimSpace(raw[k])
		if v == "" {
			return ""
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, "\x1f")
}

func formatOtherRows(rows []int, current int) string {
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		if r == current {
			continue
		}
		parts = append(parts, fmt.Sprintf("%d", r))
	}
	if len(parts) > 3 {
		return strings.Join(parts[:3], "、") + " 等"
	}
	return strings.Join(parts, "、")
}

// ExcludeImportRowsByExcelRow 从有效行中剔除指定 Excel 行号
func ExcludeImportRowsByExcelRow(rows []map[string]interface{}, excelRowNums []int, exclude map[int]struct{}) ([]map[string]interface{}, []int) {
	if len(exclude) == 0 {
		return rows, excelRowNums
	}
	outRows := make([]map[string]interface{}, 0, len(rows))
	outNums := make([]int, 0, len(rows))
	for i, row := range rows {
		excelRow := 0
		if i < len(excelRowNums) {
			excelRow = excelRowNums[i]
		}
		if _, skip := exclude[excelRow]; skip {
			continue
		}
		outRows = append(outRows, row)
		outNums = append(outNums, excelRow)
	}
	return outRows, outNums
}

// DuplicateErrorRows 提取错误涉及的 Excel 行号集合
func DuplicateErrorRows(errs []ImportRowError) map[int]struct{} {
	out := make(map[int]struct{})
	for _, e := range errs {
		if e.Row > 0 {
			out[e.Row] = struct{}{}
		}
	}
	return out
}
