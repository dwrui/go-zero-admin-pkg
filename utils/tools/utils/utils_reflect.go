package utils

import (
	"reflect"
)

// CanCallIsNil 可以调用reflect.Value的IsNil方法。
// 它可以避免 reflect.Value.IsNil 引发的 panic。
func CanCallIsNil(v interface{}) bool {
	rv, ok := v.(reflect.Value)
	if !ok {
		return false
	}
	switch rv.Kind() {
	case reflect.Interface, reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return true
	default:
		return false
	}
}
