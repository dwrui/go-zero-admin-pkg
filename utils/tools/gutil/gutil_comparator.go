package gutil

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"strings"
)

// Comparator是一个函数，用于比较a和b，并将结果以整数的形式返回
//
// Should return a number:
//
//	negative , if a < b
//	zero     , if a == b
//	positive , if a > b
type Comparator func(a, b interface{}) int

// ComparatorString 提供了一种对字符串进行快速比较的方法.
func ComparatorString(a, b interface{}) int {
	return strings.Compare(gconv.String(a), gconv.String(b))
}

// ComparatorInt 提供了一个针对 int 的基本比较方法。
func ComparatorInt(a, b interface{}) int {
	return gconv.Int(a) - gconv.Int(b)
}

// ComparatorInt8 提供对 int8 的基本比较功能。
func ComparatorInt8(a, b interface{}) int {
	return int(gconv.Int8(a) - gconv.Int8(b))
}

// ComparatorInt16 提供对 int16 的基本比较功能。
func ComparatorInt16(a, b interface{}) int {
	return int(gconv.Int16(a) - gconv.Int16(b))
}

// ComparatorInt32 提供对 int32 的基本比较功能。
func ComparatorInt32(a, b interface{}) int {
	return int(gconv.Int32(a) - gconv.Int32(b))
}

// ComparatorInt64 提供对 int64 的基本比较功能。
func ComparatorInt64(a, b interface{}) int {
	return int(gconv.Int64(a) - gconv.Int64(b))
}

// ComparatorUint 提供对 uint 的基本比较功能。
func ComparatorUint(a, b interface{}) int {
	return int(gconv.Uint(a) - gconv.Uint(b))
}

// ComparatorUint8 提供对 uint8 的基本比较功能。
func ComparatorUint8(a, b interface{}) int {
	return int(gconv.Uint8(a) - gconv.Uint8(b))
}

// ComparatorUint16 提供对 uint16 的基本比较功能。
func ComparatorUint16(a, b interface{}) int {
	return int(gconv.Uint16(a) - gconv.Uint16(b))
}

// ComparatorUint32 提供对 uint32 的基本比较功能。
func ComparatorUint32(a, b interface{}) int {
	return int(gconv.Uint32(a) - gconv.Uint32(b))
}

// ComparatorUint64 提供对 uint64 的基本比较功能。
func ComparatorUint64(a, b interface{}) int {
	return int(gconv.Uint64(a) - gconv.Uint64(b))
}

// ComparatorFloat32 提供对 float32 的基本比较功能。
func ComparatorFloat32(a, b interface{}) int {
	aFloat := gconv.Float32(a)
	bFloat := gconv.Float32(b)
	if aFloat == bFloat {
		return 0
	}
	if aFloat > bFloat {
		return 1
	}
	return -1
}

// ComparatorFloat64 提供对 float64 的基本比较功能。
func ComparatorFloat64(a, b interface{}) int {
	aFloat := gconv.Float64(a)
	bFloat := gconv.Float64(b)
	if aFloat == bFloat {
		return 0
	}
	if aFloat > bFloat {
		return 1
	}
	return -1
}

// ComparatorByte 提供对 byte 的基本比较功能。
func ComparatorByte(a, b interface{}) int {
	return int(gconv.Byte(a) - gconv.Byte(b))
}

// ComparatorRune 提供对 rune 的基本比较功能。
func ComparatorRune(a, b interface{}) int {
	return int(gconv.Rune(a) - gconv.Rune(b))
}

// ComparatorTime 提供对 time.Time 的基本比较功能。
func ComparatorTime(a, b interface{}) int {
	aTime := gconv.Time(a)
	bTime := gconv.Time(b)
	switch {
	case aTime.After(bTime):
		return 1
	case aTime.Before(bTime):
		return -1
	default:
		return 0
	}
}
