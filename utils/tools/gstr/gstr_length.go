package gstr

import "unicode/utf8"

// LenRune 返回字符串 `str` 的 Unicode 码点数量。
func LenRune(str string) int {
	return utf8.RuneCountInString(str)
}
