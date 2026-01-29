// package empty 提供了一些用于检查空/零变量的函数。
package empty

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/reflection"
	"reflect"
	"time"
)

// iString 用于 String() 的类型断言 interface。
type iString interface {
	String() string
}

// iInterfaces 用于 Interfaces() 的类型断言 interface。
type iInterfaces interface {
	Interfaces() []interface{}
}

// iMapStrAny 用于 MapStrAny() 的类型断言 interface。
type iMapStrAny interface {
	MapStrAny() map[string]interface{}
}

type iTime interface {
	Date() (year int, month time.Month, day int)
	IsZero() bool
}

// IsEmpty 检查给定的 `value` 是否为空。
// 如果 `value` 是以下类型之一，它将返回 true：0, nil, false, "", len(slice/map/chan) == 0,
// 否则它将返回 false。
//
// 参数 `traceSource` 用于在给定 `value` 是指针类型且指向指针时，跟踪源变量是否为空。
// 如果 `traceSource` 为 true，且源变量为空，它将返回 true。
// 注意，这可能会使用反射特性，会影响性能。
// 如果 `value` 是指针类型且指向指针，且 `traceSource` 为 true，
// 它将递归跟踪指针指向的源变量是否为空。
// 如果源变量为空，它将返回 true。
// 如果 `value` 不是指针类型或指向非指针，它将直接检查是否为空。
func IsEmpty(value interface{}, traceSource ...bool) bool {
	if value == nil {
		return true
	}
	//首先，它使用断言检查变量是否为常见类型，以提高性能，// 然后使用反射检查其他类型。
	switch result := value.(type) {
	case int:
		return result == 0
	case int8:
		return result == 0
	case int16:
		return result == 0
	case int32:
		return result == 0
	case int64:
		return result == 0
	case uint:
		return result == 0
	case uint8:
		return result == 0
	case uint16:
		return result == 0
	case uint32:
		return result == 0
	case uint64:
		return result == 0
	case float32:
		return result == 0
	case float64:
		return result == 0
	case bool:
		return !result
	case string:
		return result == ""
	case []byte:
		return len(result) == 0
	case []rune:
		return len(result) == 0
	case []int:
		return len(result) == 0
	case []string:
		return len(result) == 0
	case []float32:
		return len(result) == 0
	case []float64:
		return len(result) == 0
	case map[string]interface{}:
		return len(result) == 0

	default:
		// Finally, using reflect.
		var rv reflect.Value
		if v, ok := value.(reflect.Value); ok {
			rv = v
		} else {
			rv = reflect.ValueOf(value)
			if IsNil(rv) {
				return true
			}

			// =========================
			// Common interfaces checks.
			// =========================
			if f, ok := value.(iTime); ok {
				if f == (*time.Time)(nil) {
					return true
				}
				return f.IsZero()
			}
			if f, ok := value.(iString); ok {
				if f == nil {
					return true
				}
				return f.String() == ""
			}
			if f, ok := value.(iInterfaces); ok {
				if f == nil {
					return true
				}
				return len(f.Interfaces()) == 0
			}
			if f, ok := value.(iMapStrAny); ok {
				if f == nil {
					return true
				}
				return len(f.MapStrAny()) == 0
			}
		}

		switch rv.Kind() {
		case reflect.Bool:
			return !rv.Bool()

		case
			reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64:
			return rv.Int() == 0

		case
			reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64,
			reflect.Uintptr:
			return rv.Uint() == 0

		case
			reflect.Float32,
			reflect.Float64:
			return rv.Float() == 0

		case reflect.String:
			return rv.Len() == 0

		case reflect.Struct:
			var fieldValueInterface interface{}
			for i := 0; i < rv.NumField(); i++ {
				fieldValueInterface, _ = reflection.ValueToInterface(rv.Field(i))
				if !IsEmpty(fieldValueInterface) {
					return false
				}
			}
			return true

		case
			reflect.Chan,
			reflect.Map,
			reflect.Slice,
			reflect.Array:
			return rv.Len() == 0

		case reflect.Ptr:
			if len(traceSource) > 0 && traceSource[0] {
				return IsEmpty(rv.Elem())
			}
			return rv.IsNil()

		case
			reflect.Func,
			reflect.Interface,
			reflect.UnsafePointer:
			return rv.IsNil()

		case reflect.Invalid:
			return true

		default:
			return false
		}
	}
}

// IsNil 函数用于检查给定的 `value` 是否为 nil，尤其是对于 interface{} 类型的值。
// 如果给定的`value`是指针类型，则参数`traceSource`用于追踪到源变量
// 这也指向一个指针。如果`traceSource`为true时源为nil，则返回nil。
// 注意，它可能会使用反射特性，这会对性能产生一定影响。
func IsNil(value interface{}, traceSource ...bool) bool {
	if value == nil {
		return true
	}
	var rv reflect.Value
	if v, ok := value.(reflect.Value); ok {
		rv = v
	} else {
		rv = reflect.ValueOf(value)
	}
	switch rv.Kind() {
	case reflect.Chan,
		reflect.Map,
		reflect.Slice,
		reflect.Func,
		reflect.Interface,
		reflect.UnsafePointer:
		return !rv.IsValid() || rv.IsNil()

	case reflect.Ptr:
		if len(traceSource) > 0 && traceSource[0] {
			for rv.Kind() == reflect.Ptr {
				rv = rv.Elem()
			}
			if !rv.IsValid() {
				return true
			}
			if rv.Kind() == reflect.Ptr {
				return rv.IsNil()
			}
		} else {
			return !rv.IsValid() || rv.IsNil()
		}

	default:
		return false
	}
	return false
}
