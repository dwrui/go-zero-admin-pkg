package gstr

import (
	"bytes"
	"strings"
	"unicode"
)

// Count 返回字符串 `s` 中 `substr` 出现的次数。
// 如果 `substr` 未在 `s` 中找到，则返回 0。
func Count(s, substr string) int {
	return strings.Count(s, substr)
}

// CountI 返回字符串 `s` 中 `substr` 出现的次数，不区分大小写。
// 如果 `substr` 未在 `s` 中找到，则返回 0。
func CountI(s, substr string) int {
	return strings.Count(ToLower(s), ToLower(substr))
}

// CountWords 返回字符串 `str` 中单词的数量。
// 它考虑参数 `str` 为 Unicode 字符串。
func CountWords(str string) map[string]int {
	m := make(map[string]int)
	buffer := bytes.NewBuffer(nil)
	for _, r := range []rune(str) {
		if unicode.IsSpace(r) {
			if buffer.Len() > 0 {
				m[buffer.String()]++
				buffer.Reset()
			}
		} else {
			buffer.WriteRune(r)
		}
	}
	if buffer.Len() > 0 {
		m[buffer.String()]++
	}
	return m
}

// CountChars 返回字符串 `str` 中字符的数量。
// 如果参数 `noSpace` 为 true，则不统计空格字符。
// 它考虑参数 `str` 为 Unicode 字符串。
func CountChars(str string, noSpace ...bool) map[string]int {
	m := make(map[string]int)
	countSpace := true
	if len(noSpace) > 0 && noSpace[0] {
		countSpace = false
	}
	for _, r := range []rune(str) {
		if !countSpace && unicode.IsSpace(r) {
			continue
		}
		m[string(r)]++
	}
	return m
}
