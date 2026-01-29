// gtype包提供了高性能且并发安全的基本变量类型。
package gtype

// New is alias of NewAny.
// See NewAny, NewInterface.
func New(value ...interface{}) *Any {
	return NewAny(value...)
}
