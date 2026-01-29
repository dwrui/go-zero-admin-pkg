package gstr

import (
	"bytes"
	"fmt"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/grand"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	// octReg is the regular expression object for checks octal string.
	octReg = regexp.MustCompile(`\\[0-7]{3}`)
)

// Chr 将整数 `ascii` 转换为对应的 ASCII 字符。
//
// Example:
// Chr(65) -> "A"
func Chr(ascii int) string {
	return string([]byte{byte(ascii % 256)})
}

// Ord 将字符串 `char` 的第一个字节转换为 0 到 255 之间的整数值。
//
// Example:
// Ord("A") -> 65
func Ord(char string) int {
	return int(char[0])
}

// OctStr 将字符串 `str` 中的八进制转义序列转换为其原始字符串表示。
// 例如，将 "\346\200\241" 转换为 "怡"。
//
// Example:
// OctStr("\346\200\241") -> 怡
func OctStr(str string) string {
	return octReg.ReplaceAllStringFunc(
		str,
		func(s string) string {
			i, _ := strconv.ParseInt(s[1:], 8, 0)
			return string([]byte{byte(i)})
		},
	)
}

// Reverse 将字符串 `str` 反转并返回。
//
// Example:
// Reverse("123456") -> "654321"
func Reverse(str string) string {
	runes := []rune(str)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// NumberFormat 将浮点数 `number` 格式化为字符串，包含分组的千位分隔符。
// 参数 `decimals`：设置小数部分的位数。
// 参数 `decPoint`：设置小数点分隔符。
// 参数 `thousandsSep`：设置千位分隔符。
// 参考：http://php.net/manual/en/function.number-format.php。
//
// Example:
// NumberFormat(1234.56, 2, ".", "")  -> 1234,56
// NumberFormat(1234.56, 2, ",", " ") -> 1 234,56
func NumberFormat(number float64, decimals int, decPoint, thousandsSep string) string {
	neg := false
	if number < 0 {
		number = -number
		neg = true
	}
	// Will round off
	str := fmt.Sprintf("%."+strconv.Itoa(decimals)+"F", number)
	prefix, suffix := "", ""
	if decimals > 0 {
		prefix = str[:len(str)-(decimals+1)]
		suffix = str[len(str)-decimals:]
	} else {
		prefix = str
	}
	sep := []byte(thousandsSep)
	n, l1, l2 := 0, len(prefix), len(sep)
	// thousands sep num
	c := (l1 - 1) / 3
	tmp := make([]byte, l2*c+l1)
	pos := len(tmp) - 1
	for i := l1 - 1; i >= 0; i, n, pos = i-1, n+1, pos-1 {
		if l2 > 0 && n > 0 && n%3 == 0 {
			for j := range sep {
				tmp[pos] = sep[l2-j-1]
				pos--
			}
		}
		tmp[pos] = prefix[i]
	}
	s := string(tmp)
	if decimals > 0 {
		s += decPoint + suffix
	}
	if neg {
		s = "-" + s
	}

	return s
}

// Shuffle 将字符串 `str` 随机打乱并返回。
// 它考虑参数 `str` 为 Unicode 字符串。
//
// Example:
// Shuffle("123456") -> "325164"
// Shuffle("123456") -> "231546"
// ...
func Shuffle(str string) string {
	runes := []rune(str)
	s := make([]rune, len(runes))
	for i, v := range grand.Perm(len(runes)) {
		s[i] = runes[v]
	}
	return string(s)
}

// HideStr 将字符串 `str` 的中间部分替换为 `hide`，替换的比例为 `percent`。
// 参数 `percent`：指定要替换的字符数占字符串总长度的百分比。
// 参数 `hide`：指定用于替换的字符串。
// 它考虑参数 `str` 为 Unicode 字符串。
func HideStr(str string, percent int, hide string) string {
	array := strings.Split(str, "@")
	if len(array) > 1 {
		str = array[0]
	}
	var (
		rs       = []rune(str)
		length   = len(rs)
		mid      = math.Floor(float64(length / 2))
		hideLen  = int(math.Floor(float64(length) * (float64(percent) / 100)))
		start    = int(mid - math.Floor(float64(hideLen)/2))
		hideStr  = []rune("")
		hideRune = []rune(hide)
	)
	for i := 0; i < hideLen; i++ {
		hideStr = append(hideStr, hideRune...)
	}
	buffer := bytes.NewBuffer(nil)
	buffer.WriteString(string(rs[0:start]))
	buffer.WriteString(string(hideStr))
	buffer.WriteString(string(rs[start+hideLen:]))
	if len(array) > 1 {
		buffer.WriteString("@" + array[1])
	}
	return buffer.String()
}

// Nl2Br 将字符串 `str` 中的换行符（\n\r, \r\n, \r, \n）替换为 HTML 换行标签（`<br>` 或 `<br />`）。
// 参数 `isXhtml`：如果为 true，则使用 `<br />` 标签；否则使用 `<br>` 标签。
// 它考虑参数 `str` 为 Unicode 字符串。
func Nl2Br(str string, isXhtml ...bool) string {
	r, n, runes := '\r', '\n', []rune(str)
	var br []byte
	if len(isXhtml) > 0 && isXhtml[0] {
		br = []byte("<br />")
	} else {
		br = []byte("<br>")
	}
	skip := false
	length := len(runes)
	var buf bytes.Buffer
	for i, v := range runes {
		if skip {
			skip = false
			continue
		}
		switch v {
		case n, r:
			if (i+1 < length) && ((v == r && runes[i+1] == n) || (v == n && runes[i+1] == r)) {
				buf.Write(br)
				skip = true
				continue
			}
			buf.Write(br)
		default:
			buf.WriteRune(v)
		}
	}
	return buf.String()
}

// WordWrap 将字符串 `str` 按指定宽度 `width` 进行换行，支持英文和中文标点符号的换行。
// 参数 `br`：指定换行符，默认使用 `\n`。
// TODO：支持自定义换行参数，参考：http://php.net/manual/en/function.wordwrap.php。
func WordWrap(str string, width int, br string) string {
	if br == "" {
		br = "\n"
	}
	var (
		current           int
		wordBuf, spaceBuf bytes.Buffer
		init              = make([]byte, 0, len(str))
		buf               = bytes.NewBuffer(init)
		strRunes          = []rune(str)
	)
	for _, char := range strRunes {
		switch {
		case char == '\n':
			if wordBuf.Len() == 0 {
				if current+spaceBuf.Len() > width {
					current = 0
				} else {
					current += spaceBuf.Len()
					_, _ = spaceBuf.WriteTo(buf)
				}
				spaceBuf.Reset()
			} else {
				current += spaceBuf.Len() + wordBuf.Len()
				_, _ = spaceBuf.WriteTo(buf)
				spaceBuf.Reset()
				_, _ = wordBuf.WriteTo(buf)
				wordBuf.Reset()
			}
			buf.WriteRune(char)
			current = 0

		case unicode.IsSpace(char):
			if spaceBuf.Len() == 0 || wordBuf.Len() > 0 {
				current += spaceBuf.Len() + wordBuf.Len()
				_, _ = spaceBuf.WriteTo(buf)
				spaceBuf.Reset()
				_, _ = wordBuf.WriteTo(buf)
				wordBuf.Reset()
			}
			spaceBuf.WriteRune(char)

		case isPunctuation(char):
			wordBuf.WriteRune(char)
			if spaceBuf.Len() == 0 || wordBuf.Len() > 0 {
				current += spaceBuf.Len() + wordBuf.Len()
				_, _ = spaceBuf.WriteTo(buf)
				spaceBuf.Reset()
				_, _ = wordBuf.WriteTo(buf)
				wordBuf.Reset()
			}

		default:
			wordBuf.WriteRune(char)
			if current+spaceBuf.Len()+wordBuf.Len() > width && wordBuf.Len() < width {
				buf.WriteString(br)
				current = 0
				spaceBuf.Reset()
			}
		}
	}

	if wordBuf.Len() == 0 {
		if current+spaceBuf.Len() <= width {
			_, _ = spaceBuf.WriteTo(buf)
		}
	} else {
		_, _ = spaceBuf.WriteTo(buf)
		_, _ = wordBuf.WriteTo(buf)
	}
	return buf.String()
}

func isPunctuation(char int32) bool {
	switch char {
	// English Punctuations.
	case ';', '.', ',', ':', '~':
		return true
	// Chinese Punctuations.
	case '；', '，', '。', '：', '？', '！', '…', '、':
		return true
	default:
		return false
	}
}
