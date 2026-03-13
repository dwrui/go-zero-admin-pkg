package gerror

import (
	"fmt"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gcode"
)

// New 创建并返回一个从给定文本格式化的错误。
func New(text string) error {
	return &Error{
		stack: callers(),
		text:  text,
		code:  gcode.CodeNil,
	}
}

// Newf 返回一个格式化错误，其格式为给定的格式和参数。
func Newf(format string, args ...interface{}) error {
	return &Error{
		stack: callers(),
		text:  fmt.Sprintf(format, args...),
		code:  gcode.CodeNil,
	}
}

// NewSkip 创建并返回一个从给定文本格式化的错误。
// 参数 `skip` 指定了要跳过的栈调用者数量。
func NewSkip(skip int, text string) error {
	return &Error{
		stack: callers(skip),
		text:  text,
		code:  gcode.CodeNil,
	}
}

// NewSkipf 返回一个错误，该错误按照给定的格式和参数进行格式化。
// 参数`skip`指定跳过的堆栈调用者数量。
func NewSkipf(skip int, format string, args ...interface{}) error {
	return &Error{
		stack: callers(skip),
		text:  fmt.Sprintf(format, args...),
		code:  gcode.CodeNil,
	}
}

// Wrap 函数用于将错误与文本进行包装。如果给定的 err 为 nil，则该函数返回 nil。
// 注意，它不会丢失被包装错误的错误代码，因为它继承了该错误代码
func Wrap(err error, text string) error {
	if err == nil {
		return nil
	}
	return &Error{
		error: err,
		stack: callers(),
		text:  text,
		code:  Code(err),
	}
}

// Wrapf 在被调用时返回一个错误，该错误带有堆栈跟踪和格式说明符。
// 如果给定的`err`为空，则返回空。
// 注意，它不会丢失被包装错误的错误代码，因为它继承了该错误代码
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return &Error{
		error: err,
		stack: callers(),
		text:  fmt.Sprintf(format, args...),
		code:  Code(err),
	}
}

// WrapSkip 函数将错误与文本进行包装。如果给定的 err 为 nil，则该函数返回 nil。
// 参数`skip`指定跳过的堆栈调用者数量。
// 注意，它不会丢失被包装错误的错误代码，因为它继承了该错误代码。
func WrapSkip(skip int, err error, text string) error {
	if err == nil {
		return nil
	}
	return &Error{
		error: err,
		stack: callers(skip),
		text:  text,
		code:  Code(err),
	}
}

// WrapSkipf 函数使用给定格式和参数格式化的文本对错误进行包装。如果给定的 err 为 nil，则该函数返回 nil。
// 参数`skip`指定跳过的堆栈调用者数量。
// 注意，它不会丢失被包装错误的错误代码，因为它继承了该错误代码。
func WrapSkipf(skip int, err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return &Error{
		error: err,
		stack: callers(skip),
		text:  fmt.Sprintf(format, args...),
		code:  Code(err),
	}
}
