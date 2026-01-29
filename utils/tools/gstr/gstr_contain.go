package gstr

import "strings"

// Contains 将字符串 `str` 中是否包含子字符串 `substr`。
// 如果 `str` 包含 `substr`，则返回 true；否则返回 false。
func Contains(str, substr string) bool {
	return strings.Contains(str, substr)
}

// ContainsI 将字符串 `str` 中是否包含子字符串 `substr`，不区分大小写。
// 如果 `str` 包含 `substr`，则返回 true；否则返回 false。
func ContainsI(str, substr string) bool {
	return PosI(str, substr) != -1
}

// ContainsAny 将字符串 `s` 中是否包含 `chars` 中的任意一个 Unicode 代码点。
// 如果 `s` 包含 `chars` 中的任意一个代码点，则返回 true；否则返回 false。
func ContainsAny(s, chars string) bool {
	return strings.ContainsAny(s, chars)
}
