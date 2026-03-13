package gvar

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gtime"
	"time"
)

// Time converts and returns `v` as time.Time.
// The parameter `format` specifies the format of the time string using gtime,
// eg: Y-m-d H:i:s.
func (v *Var) Time(format ...string) time.Time {
	return gconv.Time(v.Val(), format...)
}

// Duration converts and returns `v` as time.Duration.
// If value of `v` is string, then it uses time.ParseDuration for conversion.
func (v *Var) Duration() time.Duration {
	return gconv.Duration(v.Val())
}

// GTime converts and returns `v` as *gtime.Time.
// The parameter `format` specifies the format of the time string using gtime,
// eg: Y-m-d H:i:s.
func (v *Var) GTime(format ...string) *gtime.Time {
	return gconv.GTime(v.Val(), format...)
}
