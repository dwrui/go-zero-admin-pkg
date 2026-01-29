package gstr

// SearchArray 搜索字符串 `s` 在字符串切片 `a` 中的位置（区分大小写），
// 如果找到则返回其在 `a` 中的索引，否则返回 -1。
func SearchArray(a []string, s string) int {
	for i, v := range a {
		if s == v {
			return i
		}
	}
	return NotFoundIndex
}

// InArray 检查字符串 `s` 是否在字符串切片 `a` 中。
func InArray(a []string, s string) bool {
	return SearchArray(a, s) != NotFoundIndex
}

// PrefixArray 为 'array' 中的每个项添加 'prefix' 字符串.
//
// Example:
// PrefixArray(["a","b"], "gf_") -> ["gf_a", "gf_b"]
func PrefixArray(array []string, prefix string) {
	for k, v := range array {
		array[k] = prefix + v
	}
}
