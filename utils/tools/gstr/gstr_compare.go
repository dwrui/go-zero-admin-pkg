package gstr

import "strings"

// Compare 将字符串 `a` 和 `b` 按字典顺序进行比较。
// 如果 `a` 等于 `b`，则返回 0；如果 `a` 小于 `b`，则返回 -1；如果 `a` 大于 `b`，则返回 +1。
func Compare(a, b string) int {
	return strings.Compare(a, b)
}

// Equal 将字符串 `a` 和 `b` 进行不区分大小写的 Unicode 大小写折叠比较。
// 如果 `a` 等于 `b`，则返回 true；否则返回 false。
func Equal(a, b string) bool {
	return strings.EqualFold(a, b)
}
