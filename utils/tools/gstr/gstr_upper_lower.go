package gstr

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/utils"
	"strings"
)

// ToLower 返回字符串 `s` 的小写副本。
func ToLower(s string) string {
	return strings.ToLower(s)
}

// ToUpper 返回字符串 `s` 的大写副本。
func ToUpper(s string) string {
	return strings.ToUpper(s)
}

// UcFirst 返回字符串 `s` 的副本，其中第一个字母映射为大写。
func UcFirst(s string) string {
	return utils.UcFirst(s)
}

// LcFirst 返回字符串 `s` 的副本，其中第一个字母映射为小写。
func LcFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	if IsLetterUpper(s[0]) {
		return string(s[0]+32) + s[1:]
	}
	return s
}

// UcWords 返回字符串 `str` 的副本，其中每个单词的第一个字母都映射为大写。
func UcWords(str string) string {
	return strings.Title(str)
}

// IsLetterLower 测试给定的字节 `b` 是否为小写字母。
func IsLetterLower(b byte) bool {
	return utils.IsLetterLower(b)
}

// IsLetterUpper 测试给定的字节 `b` 是否为大写字母。
func IsLetterUpper(b byte) bool {
	return utils.IsLetterUpper(b)
}
