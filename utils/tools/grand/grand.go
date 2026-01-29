// Package Grand 提供高性能的随机字节/数字/字符串生成功能。
package grand

import (
	"encoding/binary"
	"time"
)

var (
	letters    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" // 52
	symbols    = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"                   // 32
	digits     = "0123456789"                                           // 10
	characters = letters + digits + symbols                             // 94
)

// Intn 返回一个 int 类型的随机数，该随机数在 0 到 max 之间：[0, max)。
//
// 注意：
// 1. `max` 只能大于 0，否则直接返回 `max`；
// 2. 结果大于等于 0，小于 `max`；
// 3. 结果是 32 位整数，小于 math.MaxUint32。
func Intn(max int) int {
	if max <= 0 {
		return max
	}
	n := int(binary.LittleEndian.Uint32(<-bufferChan)) % max
	if n < 0 {
		return -n
	}
	return n
}

// B 返回一个随机字节切片，长度为 `n`。
func B(n int) []byte {
	if n <= 0 {
		return nil
	}
	i := 0
	b := make([]byte, n)
	for {
		copy(b[i:], <-bufferChan)
		i += 4
		if i >= n {
			break
		}
	}
	return b
}

// N 返回一个随机 int 类型的数，该数在 min 和 max 之间：[min, max]。
// 注意：
// 1. `min` 和 `max` 也支持负数；
// 2. 如果 `min` 大于等于 `max`，则直接返回 `min`；
// 3. 结果是 32 位整数，小于 math.MaxUint32。
func N(min, max int) int {
	if min >= max {
		return min
	}
	if min >= 0 {
		return Intn(max-min+1) + min
	}
	// As `Intn` dose not support negative number,
	// so we should first shift the value to right,
	// then call `Intn` to produce the random number,
	// and finally shift the result back to left.
	return Intn(max+(0-min)+1) - (0 - min)
}

// S 返回一个包含数字和字母的随机字符串，其长度为 'n'。
// 可选参数“symbols”指定结果是否可以包含符号。
// 默认情况下是 false。
func S(n int, symbols ...bool) string {
	if n <= 0 {
		return ""
	}
	var (
		b           = make([]byte, n)
		numberBytes = B(n)
	)
	for i := range b {
		if len(symbols) > 0 && symbols[0] {
			b[i] = characters[numberBytes[i]%94]
		} else {
			b[i] = characters[numberBytes[i]%62]
		}
	}
	return string(b)
}

// D 返回一个随机 time.Duration 类型的数，该数在 min 和 max 之间：[min, max]。
// 注意：
// 1. `min` 和 `max` 也支持负数；
// 2. 如果 `min` 大于等于 `max`，则直接返回 `min`；
// 3. 结果是 32 位整数，小于 math.MaxUint32。
func D(min, max time.Duration) time.Duration {
	multiple := int64(1)
	if min != 0 {
		for min%10 == 0 {
			multiple *= 10
			min /= 10
			max /= 10
		}
	}
	n := int64(N(int(min), int(max)))
	return time.Duration(n * multiple)
}

// Str 返回一个随机字符串，该字符串包含从给定字符串 `s` 中随机选择的 `n` 个字符。
// 它还支持 Unicode 字符串，如中文、俄语、日语等。
func Str(s string, n int) string {
	if n <= 0 {
		return ""
	}
	var (
		b     = make([]rune, n)
		runes = []rune(s)
	)
	if len(runes) <= 255 {
		numberBytes := B(n)
		for i := range b {
			b[i] = runes[int(numberBytes[i])%len(runes)]
		}
	} else {
		for i := range b {
			b[i] = runes[Intn(len(runes))]
		}
	}
	return string(b)
}

// Digits 返回一个随机字符串，该字符串仅包含数字，长度为 `n`。
func Digits(n int) string {
	if n <= 0 {
		return ""
	}
	var (
		b           = make([]byte, n)
		numberBytes = B(n)
	)
	for i := range b {
		b[i] = digits[numberBytes[i]%10]
	}
	return string(b)
}

// Letters 返回一个随机字符串，该字符串仅包含字母，长度为 `n`。
func Letters(n int) string {
	if n <= 0 {
		return ""
	}
	var (
		b           = make([]byte, n)
		numberBytes = B(n)
	)
	for i := range b {
		b[i] = letters[numberBytes[i]%52]
	}
	return string(b)
}

// Symbols 返回一个随机字符串，该字符串仅包含符号，长度为 `n`。
func Symbols(n int) string {
	if n <= 0 {
		return ""
	}
	var (
		b           = make([]byte, n)
		numberBytes = B(n)
	)
	for i := range b {
		b[i] = symbols[numberBytes[i]%32]
	}
	return string(b)
}

// Perm 返回一个包含 n 个 int 类型随机数的切片，这些随机数是 [0,n) 之间的伪随机排列。
// TODO 对于大型切片的生产，性能可以改进。
func Perm(n int) []int {
	m := make([]int, n)
	for i := 0; i < n; i++ {
		j := Intn(i + 1)
		m[i] = m[j]
		m[j] = i
	}
	return m
}

// Meet 返回一个 bool 值，该值表示是否满足给定的概率 `num`/`total`。
func Meet(num, total int) bool {
	return Intn(total) < num
}

// MeetProb 返回一个 bool 值，该值表示是否满足给定的概率 `prob`。
func MeetProb(prob float32) bool {
	return Intn(1e7) < int(prob*1e7)
}
