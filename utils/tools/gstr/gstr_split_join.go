package gstr

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/utils"
	"strings"
)

// Split 将字符串 str 分割成字符串 “delimiter”，生成数组。
func Split(str, delimiter string) []string {
	return strings.Split(str, delimiter)
}

// SplitAndTrim 将字符串 str 分割成字符串 “delimiter”，生成数组，
// 并对数组中的每个元素调用 Trim 函数。
// 它会忽略在 Trim 后为空的元素。
func SplitAndTrim(str, delimiter string, characterMask ...string) []string {
	return utils.SplitAndTrim(str, delimiter, characterMask...)
}

// Join 将数组 array 中的元素连接起来，使用字符串 sep 作为分隔符。
func Join(array []string, sep string) string {
	return strings.Join(array, sep)
}

// JoinAny 将数组 array 中的元素连接起来，使用字符串 sep 作为分隔符。
//
// 参数 array 可以是任何类型的切片，它会被转换为字符串数组。
func JoinAny(array interface{}, sep string) string {
	return strings.Join(gconv.Strings(array), sep)
}

// Explode 将字符串 str 分割成字符串 “delimiter”，生成数组。
// 它与 Split 函数相同。
// 请参阅 http://php.net/manual/en/function.explode.php。
func Explode(delimiter, str string) []string {
	return Split(str, delimiter)
}

// Implode 将数组 pieces 中的元素连接起来，使用字符串 glue 作为分隔符。
// 它与 Join 函数相同。
// 请参阅 http://php.net/manual/en/function.implode.php。
func Implode(glue string, pieces []string) string {
	return strings.Join(pieces, glue)
}

// ChunkSplit 将字符串拆分成更小的块.
// 可以用来将字符串拆分成更小的块，这对处理 BASE64 字符串输出非常有用，
// 例如将 BASE64 字符串输出转换为符合 RFC 2045 语义的格式。
// 它每 chunkLen 个字符插入 end。
// 它将参数 body 和 end 视为 Unicode 字符串。
func ChunkSplit(body string, chunkLen int, end string) string {
	if end == "" {
		end = "\r\n"
	}
	runes, endRunes := []rune(body), []rune(end)
	l := len(runes)
	if l <= 1 || l < chunkLen {
		return body + end
	}
	ns := make([]rune, 0, len(runes)+len(endRunes))
	for i := 0; i < l; i += chunkLen {
		if i+chunkLen > l {
			ns = append(ns, runes[i:]...)
		} else {
			ns = append(ns, runes[i:i+chunkLen]...)
		}
		ns = append(ns, endRunes...)
	}
	return string(ns)
}

// Fields 将字符串 str 分割成单词，生成数组。
// 它会忽略在 Trim 后为空的元素。
func Fields(str string) []string {
	return strings.Fields(str)
}
