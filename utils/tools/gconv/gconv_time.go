package gconv

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gtime"
	"time"
)

// Time converts `any` to time.Time.
func Time(any any, format ...string) time.Time {
	t, _ := defaultConverter.Time(any, format...)
	return t
}

// Duration converts `any` to time.Duration.
// If `any` is string, then it uses time.ParseDuration to convert it.
// If `any` is numeric, then it converts `any` as nanoseconds.
func Duration(any any) time.Duration {
	d, _ := defaultConverter.Duration(any)
	return d
}

// GTime converts `any` to *gtime.Time.
// The parameter `format` can be used to specify the format of `any`.
// It returns the converted value that matched the first format of the formats slice.
// If no `format` given, it converts `any` using gtime.NewFromTimeStamp if `any` is numeric,
// or using gtime.StrToTime if `any` is string.
func GTime(any any, format ...string) *gtime.Time {
	t, _ := defaultConverter.GTime(any, format...)
	return t
}
