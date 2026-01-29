package gstr

import "strings"

// Repeat 返回由字符串 `input` 重复 `multiplier` 次组成的新字符串。
//
// 示例：
// Repeat("a", 3) -> "aaa"
func Repeat(input string, multiplier int) string {
	return strings.Repeat(input, multiplier)
}
