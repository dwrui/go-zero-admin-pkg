package gstr

import "strings"

// Pos 返回字符串 `haystack` 中第一次出现 `needle` 的位置，从 `startOffset` 开始搜索。
// 如果未找到，则返回 -1。
func Pos(haystack, needle string, startOffset ...int) int {
	length := len(haystack)
	offset := 0
	if len(startOffset) > 0 {
		offset = startOffset[0]
	}
	if length == 0 || offset > length || -offset > length {
		return -1
	}
	if offset < 0 {
		offset += length
	}
	pos := strings.Index(haystack[offset:], needle)
	if pos == NotFoundIndex {
		return NotFoundIndex
	}
	return pos + offset
}

// PosRune 返回字符串 `haystack` 中第一次出现 `needle` 的位置，从 `startOffset` 开始搜索，考虑 Unicode 字符。
// 如果未找到，则返回 -1。
func PosRune(haystack, needle string, startOffset ...int) int {
	pos := Pos(haystack, needle, startOffset...)
	if pos < 3 {
		return pos
	}
	return len([]rune(haystack[:pos]))
}

// PosI 返回字符串 `haystack` 中第一次出现 `needle` 的位置，从 `startOffset` 开始搜索，不区分大小写。
// 如果未找到，则返回 -1。
func PosI(haystack, needle string, startOffset ...int) int {
	length := len(haystack)
	offset := 0
	if len(startOffset) > 0 {
		offset = startOffset[0]
	}
	if length == 0 || offset > length || -offset > length {
		return -1
	}

	if offset < 0 {
		offset += length
	}
	pos := strings.Index(strings.ToLower(haystack[offset:]), strings.ToLower(needle))
	if pos == -1 {
		return -1
	}
	return pos + offset
}

// PosIRune 返回字符串 `haystack` 中第一次出现 `needle` 的位置，从 `startOffset` 开始搜索，不区分大小写，考虑 Unicode 字符。
// 如果未找到，则返回 -1。
func PosIRune(haystack, needle string, startOffset ...int) int {
	pos := PosI(haystack, needle, startOffset...)
	if pos < 3 {
		return pos
	}
	return len([]rune(haystack[:pos]))
}

// PosR 返回字符串 `haystack` 中最后一次出现 `needle` 的位置，从 `startOffset` 开始搜索。
// 如果未找到，则返回 -1。
func PosR(haystack, needle string, startOffset ...int) int {
	offset := 0
	if len(startOffset) > 0 {
		offset = startOffset[0]
	}
	pos, length := 0, len(haystack)
	if length == 0 || offset > length || -offset > length {
		return -1
	}

	if offset < 0 {
		haystack = haystack[:offset+length+1]
	} else {
		haystack = haystack[offset:]
	}
	pos = strings.LastIndex(haystack, needle)
	if offset > 0 && pos != -1 {
		pos += offset
	}
	return pos
}

// PosRRune 返回字符串 `haystack` 中最后一次出现 `needle` 的位置，从 `startOffset` 开始搜索，考虑 Unicode 字符。
// 如果未找到，则返回 -1。
func PosRRune(haystack, needle string, startOffset ...int) int {
	pos := PosR(haystack, needle, startOffset...)
	if pos < 3 {
		return pos
	}
	return len([]rune(haystack[:pos]))
}

// PosRI 返回字符串 `haystack` 中最后一次出现 `needle` 的位置，从 `startOffset` 开始搜索，不区分大小写。
// 如果未找到，则返回 -1。
func PosRI(haystack, needle string, startOffset ...int) int {
	offset := 0
	if len(startOffset) > 0 {
		offset = startOffset[0]
	}
	pos, length := 0, len(haystack)
	if length == 0 || offset > length || -offset > length {
		return -1
	}

	if offset < 0 {
		haystack = haystack[:offset+length+1]
	} else {
		haystack = haystack[offset:]
	}
	pos = strings.LastIndex(strings.ToLower(haystack), strings.ToLower(needle))
	if offset > 0 && pos != -1 {
		pos += offset
	}
	return pos
}

// PosRIRune 返回字符串 `haystack` 中最后一次出现 `needle` 的位置，从 `startOffset` 开始搜索，不区分大小写，考虑 Unicode 字符。
// 如果未找到，则返回 -1。
func PosRIRune(haystack, needle string, startOffset ...int) int {
	pos := PosRI(haystack, needle, startOffset...)
	if pos < 3 {
		return pos
	}
	return len([]rune(haystack[:pos]))
}
