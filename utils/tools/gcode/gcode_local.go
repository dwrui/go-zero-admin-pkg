package gcode

import "fmt"

// localCode 是 Code 接口的实现者，仅用于内部使用。
type localCode struct {
	code    int         // 错误码，通常是一个整数。
	message string      // 此错误码的简要消息。
	detail  interface{} // 作为接口类型，主要设计为错误码的扩展字段。
}

// Code 返回当前错误码的整数编号。
func (c localCode) Code() int {
	return c.code
}

// Message 返回当前错误码的简要消息。
func (c localCode) Message() string {
	return c.message
}

// Detail 返回当前错误码的详细信息，
// 主要设计为错误码的扩展字段。
func (c localCode) Detail() interface{} {
	return c.detail
}

// String 返回当前错误码作为字符串。
func (c localCode) String() string {
	if c.detail != nil {
		return fmt.Sprintf(`%d:%s %v`, c.code, c.message, c.detail)
	}
	if c.message != "" {
		return fmt.Sprintf(`%d:%s`, c.code, c.message)
	}
	return fmt.Sprintf(`%d`, c.code)
}
