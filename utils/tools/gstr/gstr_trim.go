package gstr

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/utils"
	"strings"
)

// Trim 从字符串 `str` 的开头和结尾删除空格（或其他字符）。
// 可选参数 `characterMask` 指定要删除的其他字符。
func Trim(str string, characterMask ...string) string {
	return utils.Trim(str, characterMask...)
}

// TrimStr 从字符串 `str` 的开头和结尾删除所有 `cut` 字符串。
// 可选参数 `count` 指定要删除的最大次数。
// 注意：它不会删除字符串开头或结尾的空格。
func TrimStr(str string, cut string, count ...int) string {
	return TrimLeftStr(TrimRightStr(str, cut, count...), cut, count...)
}

// TrimLeft 从字符串 `str` 的开头删除空格（或其他字符）。
// 可选参数 `characterMask` 指定要删除的其他字符。
func TrimLeft(str string, characterMask ...string) string {
	trimChars := utils.DefaultTrimChars
	if len(characterMask) > 0 {
		trimChars += characterMask[0]
	}
	return strings.TrimLeft(str, trimChars)
}

// TrimLeftStr 从字符串 `str` 的开头删除所有 `cut` 字符串。
// 可选参数 `count` 指定要删除的最大次数。
// 注意：它不会删除字符串开头的空格。
func TrimLeftStr(str string, cut string, count ...int) string {
	var (
		lenCut   = len(cut)
		cutCount = 0
	)
	for len(str) >= lenCut && str[0:lenCut] == cut {
		str = str[lenCut:]
		cutCount++
		if len(count) > 0 && count[0] != -1 && cutCount >= count[0] {
			break
		}
	}
	return str
}

// TrimRight 从字符串 `str` 的结尾删除空格（或其他字符）。
// 可选参数 `characterMask` 指定要删除的其他字符。
func TrimRight(str string, characterMask ...string) string {
	trimChars := utils.DefaultTrimChars
	if len(characterMask) > 0 {
		trimChars += characterMask[0]
	}
	return strings.TrimRight(str, trimChars)
}

// TrimRightStr 从字符串 `str` 的结尾删除所有 `cut` 字符串。
// 可选参数 `count` 指定要删除的最大次数。
// 注意：它不会删除字符串结尾的空格。
func TrimRightStr(str string, cut string, count ...int) string {
	var (
		lenStr   = len(str)
		lenCut   = len(cut)
		cutCount = 0
	)
	for lenStr >= lenCut && str[lenStr-lenCut:lenStr] == cut {
		lenStr = lenStr - lenCut
		str = str[:lenStr]
		cutCount++
		if len(count) > 0 && count[0] != -1 && cutCount >= count[0] {
			break
		}
	}
	return str
}

// TrimAll 从字符串 `str` 中删除所有空格（或其他字符）。
// 可选参数 `characterMask` 指定要删除的其他字符。
func TrimAll(str string, characterMask ...string) string {
	trimChars := utils.DefaultTrimChars
	if len(characterMask) > 0 {
		trimChars += characterMask[0]
	}
	var (
		filtered bool
		slice    = make([]rune, 0, len(str))
	)
	for _, char := range str {
		filtered = false
		for _, trimChar := range trimChars {
			if char == trimChar {
				filtered = true
				break
			}
		}
		if !filtered {
			slice = append(slice, char)
		}
	}
	return string(slice)
}

// HasPrefix 测试字符串 `s` 是否以 `prefix` 开头。
func HasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

// HasSuffix 测试字符串 `s` 是否以 `suffix` 结尾。
func HasSuffix(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}
