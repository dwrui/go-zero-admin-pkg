package gstr

import "strings"

// Str 返回 `haystack` 字符串中从第一个出现的 `needle` 到 `haystack` 结尾的部分。
//
// 此函数与 SubStr 函数相同，但为了实现与 PHP 相同的函数：http://php.net/manual/en/function.strstr.php。
//
// 示例：
// Str("av.mp4", ".") -> ".mp4"
func Str(haystack string, needle string) string {
	if needle == "" {
		return ""
	}
	pos := strings.Index(haystack, needle)
	if pos == NotFoundIndex {
		return ""
	}
	return haystack[pos+len([]byte(needle))-1:]
}

// StrEx 返回 `haystack` 字符串中从第一个出现的 `needle` 到 `haystack` 结尾的部分，不包括 `needle`。
//
// 此函数与 SubStrEx 函数相同，但为了实现与 PHP 相同的函数：http://php.net/manual/en/function.strstr.php。
//
// 示例：
// StrEx("av.mp4", ".") -> "mp4"
func StrEx(haystack string, needle string) string {
	if s := Str(haystack, needle); s != "" {
		return s[1:]
	}
	return ""
}

// StrTill 返回 `haystack` 字符串中从 `haystack` 开头到第一个出现的 `needle` 包括 `needle` 的部分。
//
// 示例：
// StrTill("av.mp4", ".") -> "av."
func StrTill(haystack string, needle string) string {
	pos := strings.Index(haystack, needle)
	if pos == NotFoundIndex || pos == 0 {
		return ""
	}
	return haystack[:pos+1]
}

// StrTillEx 返回 `haystack` 字符串中从 `haystack` 开头到第一个出现的 `needle` 不包括 `needle` 的部分。
//
// 示例：
// StrTillEx("av.mp4", ".") -> "av"
func StrTillEx(haystack string, needle string) string {
	pos := strings.Index(haystack, needle)
	if pos == NotFoundIndex || pos == 0 {
		return ""
	}
	return haystack[:pos]
}

// SubStr 返回 `str` 字符串中从 `start` 开始的 `length` 个字符。
// 如果 `length` 未指定，则返回从 `start` 开始到 `str` 结尾的所有字符。
//
// 示例：
// SubStr("123456", 1, 2) -> "23"
func SubStr(str string, start int, length ...int) (substr string) {
	strLength := len(str)
	if start < 0 {
		if -start > strLength {
			start = 0
		} else {
			start = strLength + start
		}
	} else if start > strLength {
		return ""
	}
	realLength := 0
	if len(length) > 0 {
		realLength = length[0]
		if realLength < 0 {
			if -realLength > strLength-start {
				realLength = 0
			} else {
				realLength = strLength - start + realLength
			}
		} else if realLength > strLength-start {
			realLength = strLength - start
		}
	} else {
		realLength = strLength - start
	}

	if realLength == strLength {
		return str
	} else {
		end := start + realLength
		return str[start:end]
	}
}

// SubStrRune 返回 `str` 字符串中从 `start` 开始的 `length` 个字符。
// 如果 `length` 未指定，则返回从 `start` 开始到 `str` 结尾的所有字符。
// SubStrRune 考虑参数 `str` 为 Unicode 字符串。
//
// 示例：
// SubStrRune("一起学习吧！", 2, 2) -> "学习"
func SubStrRune(str string, start int, length ...int) (substr string) {
	// Converting to []rune to support unicode.
	var (
		runes       = []rune(str)
		runesLength = len(runes)
	)

	strLength := runesLength
	if start < 0 {
		if -start > strLength {
			start = 0
		} else {
			start = strLength + start
		}
	} else if start > strLength {
		return ""
	}
	realLength := 0
	if len(length) > 0 {
		realLength = length[0]
		if realLength < 0 {
			if -realLength > strLength-start {
				realLength = 0
			} else {
				realLength = strLength - start + realLength
			}
		} else if realLength > strLength-start {
			realLength = strLength - start
		}
	} else {
		realLength = strLength - start
	}
	end := start + realLength
	if end > runesLength {
		end = runesLength
	}
	return string(runes[start:end])
}

// StrLimit 返回 `str` 字符串中从 `start` 开始的 `length` 个字符。
// 如果 `length` 未指定，则返回从 `start` 开始到 `str` 结尾的所有字符。
// 如果 `str` 的长度大于 `length`，则 `suffix` 将被追加到结果字符串中。
//
// 示例：
// StrLimit("123456", 3)      -> "123..."
// StrLimit("123456", 3, "~") -> "123~"
func StrLimit(str string, length int, suffix ...string) string {
	if len(str) < length {
		return str
	}
	suffixStr := defaultSuffixForStrLimit
	if len(suffix) > 0 {
		suffixStr = suffix[0]
	}
	return str[0:length] + suffixStr
}

// StrLimitRune 返回 `str` 字符串中从 `start` 开始的 `length` 个字符。
// 如果 `length` 未指定，则返回从 `start` 开始到 `str` 结尾的所有字符。
// 如果 `str` 的长度大于 `length`，则 `suffix` 将被追加到结果字符串中。
// StrLimitRune 考虑参数 `str` 为 Unicode 字符串。
//
// 示例：
// StrLimitRune("一起学习吧！", 2)      -> "一起..."
// StrLimitRune("一起学习吧！", 2, "~") -> "一起~"
func StrLimitRune(str string, length int, suffix ...string) string {
	runes := []rune(str)
	if len(runes) < length {
		return str
	}
	suffixStr := defaultSuffixForStrLimit
	if len(suffix) > 0 {
		suffixStr = suffix[0]
	}
	return string(runes[0:length]) + suffixStr
}

// SubStrFrom 返回 `str` 字符串中从第一个出现的 `need` 包括 `need` 到 `str` 结尾的部分。
//
// 示例：
// SubStrFrom("av.mp4", ".") -> ".mp4"
func SubStrFrom(str string, need string) (substr string) {
	pos := Pos(str, need)
	if pos < 0 {
		return ""
	}
	return str[pos:]
}

// SubStrFromEx 返回 `str` 字符串中从第一个出现的 `need` 不包括 `need` 到 `str` 结尾的部分。
//
// 示例：
// SubStrFromEx("av.mp4", ".") -> "mp4"
func SubStrFromEx(str string, need string) (substr string) {
	pos := Pos(str, need)
	if pos < 0 {
		return ""
	}
	return str[pos+len(need):]
}

// SubStrFromR 返回 `str` 字符串中从最后一个出现的 `need` 包括 `need` 到 `str` 结尾的部分。
//
// 示例：
// SubStrFromR("/dev/vda", "/") -> "/vda"
func SubStrFromR(str string, need string) (substr string) {
	pos := PosR(str, need)
	if pos < 0 {
		return ""
	}
	return str[pos:]
}

// SubStrFromREx 返回 `str` 字符串中从最后一个出现的 `need` 不包括 `need` 到 `str` 结尾的部分。
//
// 示例：
// SubStrFromREx("/dev/vda", "/") -> "vda"
func SubStrFromREx(str string, need string) (substr string) {
	pos := PosR(str, need)
	if pos < 0 {
		return ""
	}
	return str[pos+len(need):]
}
