package gstr

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/utils"
	"strings"
)

// Replace 返回字符串 `origin` 的副本，
// 其中字符串 `search` 被 `replace` 替换，区分大小写。
func Replace(origin, search, replace string, count ...int) string {
	n := -1
	if len(count) > 0 {
		n = count[0]
	}
	return strings.Replace(origin, search, replace, n)
}

// ReplaceI 返回字符串 `origin` 的副本，
// 其中字符串 `search` 被 `replace` 替换，不区分大小写。
func ReplaceI(origin, search, replace string, count ...int) string {
	n := -1
	if len(count) > 0 {
		n = count[0]
	}
	if n == 0 {
		return origin
	}
	var (
		searchLength  = len(search)
		replaceLength = len(replace)
		searchLower   = strings.ToLower(search)
		originLower   string
		pos           int
	)
	for {
		originLower = strings.ToLower(origin)
		if pos = Pos(originLower, searchLower, pos); pos != -1 {
			origin = origin[:pos] + replace + origin[pos+searchLength:]
			pos += replaceLength
			if n--; n == 0 {
				break
			}
		} else {
			break
		}
	}
	return origin
}

// ReplaceByArray 返回字符串 `origin` 的副本，
// 其中字符串 `search` 被 `replace` 替换，区分大小写。
func ReplaceByArray(origin string, array []string) string {
	for i := 0; i < len(array); i += 2 {
		if i+1 >= len(array) {
			break
		}
		origin = Replace(origin, array[i], array[i+1])
	}
	return origin
}

// ReplaceIByArray 返回字符串 `origin` 的副本，
// 其中字符串 `search` 被 `replace` 替换，不区分大小写。
func ReplaceIByArray(origin string, array []string) string {
	for i := 0; i < len(array); i += 2 {
		if i+1 >= len(array) {
			break
		}
		origin = ReplaceI(origin, array[i], array[i+1])
	}
	return origin
}

// ReplaceByMap 返回字符串 `origin` 的副本，
// 其中字符串 `search` 被 `replace` 替换，区分大小写。
func ReplaceByMap(origin string, replaces map[string]string) string {
	return utils.ReplaceByMap(origin, replaces)
}

// ReplaceIByMap 返回字符串 `origin` 的副本，
// 其中字符串 `search` 被 `replace` 替换，不区分大小写。
func ReplaceIByMap(origin string, replaces map[string]string) string {
	for k, v := range replaces {
		origin = ReplaceI(origin, k, v)
	}
	return origin
}

// ReplaceFunc 返回字符串 `origin` 的副本，
// 其中每个不重叠的子字符串，匹配给定的搜索字符串，都被函数 `f` 应用于该子字符串的结果替换。
// 函数 `f` 被每个匹配的子字符串作为参数调用，必须返回一个字符串作为替换值。
// as the replacement value.
func ReplaceFunc(origin string, search string, f func(string) string) string {
	if search == "" {
		return origin
	}
	var (
		searchLen = len(search)
		originLen = len(origin)
	)
	// 如果搜索字符串长于原字符串，则无法匹配
	if searchLen > originLen {
		return origin
	}
	var (
		result     strings.Builder
		lastMatch  int
		currentPos int
	)
	// 预先分配建设商产能以避免重新分配
	result.Grow(originLen)

	for currentPos < originLen {
		pos := Pos(origin[currentPos:], search)
		if pos == -1 {
			break
		}
		pos += currentPos
		// 追加未匹配部分
		result.WriteString(origin[lastMatch:pos])
		// 应用替换函数并追加结果
		match := origin[pos : pos+searchLen]
		result.WriteString(f(match))
		// 更新位置
		lastMatch = pos + searchLen
		currentPos = lastMatch
	}
	// 追加剩余未匹配部分
	if lastMatch < originLen {
		result.WriteString(origin[lastMatch:])
	}
	return result.String()
}

// ReplaceIFunc 返回字符串 `origin` 的副本，
// 其中每个不重叠的子字符串，匹配给定的搜索字符串，都被函数 `f` 应用于该子字符串的结果替换。
// 匹配不区分大小写。
// 函数 `f` 被每个匹配的子字符串作为参数调用，必须返回一个字符串作为替换值。
func ReplaceIFunc(origin string, search string, f func(string) string) string {
	if search == "" {
		return origin
	}
	var (
		searchLen = len(search)
		originLen = len(origin)
	)
	// 如果搜索字符串长于原字符串，则无法匹配
	if searchLen > originLen {
		return origin
	}
	var (
		result     strings.Builder
		lastMatch  int
		currentPos int
	)
	// 预先分配建设商产能以避免重新分配
	result.Grow(originLen)

	for currentPos < originLen {
		pos := PosI(origin[currentPos:], search)
		if pos == -1 {
			break
		}
		pos += currentPos
		// 追加未匹配部分
		result.WriteString(origin[lastMatch:pos])
		// 应用替换函数并追加结果
		match := origin[pos : pos+searchLen]
		result.WriteString(f(match))
		// 更新位置
		lastMatch = pos + searchLen
		currentPos = lastMatch
	}
	// 追加剩余未匹配部分
	if lastMatch < originLen {
		result.WriteString(origin[lastMatch:])
	}
	return result.String()
}
