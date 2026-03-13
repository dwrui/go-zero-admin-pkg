// Package gerror 提供了丰富的作错误功能。
//
// 对于维护者，请特别注意，
// 这个包是一个相当基础的包，不应导入额外的包
// 除了标准包和内部包，以避免循环导入。
package gerror

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gcode"
)

// IEqual 是 Equal 功能的接口。
type IEqual interface {
	Error() string
	Equal(target error) bool
}

// ICode 是 Code 功能的接口。
type ICode interface {
	Error() string
	Code() gcode.Code
}

// IStack 是 Stack 功能的接口。
type IStack interface {
	Error() string
	Stack() string
}

// ICause 是 Cause 功能的接口。
type ICause interface {
	Error() string
	Cause() error
}

// ICurrent 是 Current 功能的接口。
type ICurrent interface {
	Error() string
	Current() error
}

// IUnwrap 是 Unwrap 功能的接口。
type IUnwrap interface {
	Error() string
	Unwrap() error
}

const (
	// commaSeparatorSpace is the comma separator with space.
	commaSeparatorSpace = ", "
)
