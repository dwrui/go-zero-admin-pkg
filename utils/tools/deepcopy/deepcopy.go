// 包 deepcopy 通过反射技术实现对象的深度复制。
//
// 此包由以下地址维护：https://github.com/mohae/deepcopy
package deepcopy

import (
	"reflect"
	"time"
)

// Interface 定义了一个委托复制过程的接口。
type Interface interface {
	DeepCopy() interface{}
}

// Copy 创建 src 的一个深度拷贝。
//
// Copy 无法复制结构体中未导出的字段（字段名为小写）。
// 未导出的字段无法被Go运行时反射，因此
// 他们无法执行任何数据复制。
func Copy(src interface{}) interface{} {
	if src == nil {
		return nil
	}

	// 通过类型断言进行复制。
	switch r := src.(type) {
	case
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		complex64, complex128,
		string,
		bool:
		return r

	default:
		if v, ok := src.(Interface); ok {
			return v.DeepCopy()
		}
		var (
			original = reflect.ValueOf(src)                // 产生一个反射值
			dst      = reflect.New(original.Type()).Elem() // 产生一个与 original 类型相同的副本
		)
		// 递归复制原始值。
		copyRecursive(original, dst)
		// 返回副本作为接口。
		return dst.Interface()
	}
}

// copyRecursive 递归复制原始值到副本中。
// 它目前对可处理的类型有限制。根据需要添加。
func copyRecursive(original, cpy reflect.Value) {
	// 检查是否实现了 deepcopy.Interface 接口。
	if original.CanInterface() && original.IsValid() && !original.IsZero() {
		if copier, ok := original.Interface().(Interface); ok {
			cpy.Set(reflect.ValueOf(copier.DeepCopy()))
			return
		}
	}

	// 根据 original 的 Kind 进行复制。
	switch original.Kind() {
	case reflect.Ptr:
		// 获取被指向的值。
		originalValue := original.Elem()

		// 如果它不是有效的值，直接返回。
		if !originalValue.IsValid() {
			return
		}
		cpy.Set(reflect.New(originalValue.Type()))
		copyRecursive(originalValue, cpy.Elem())

	case reflect.Interface:
		// 如果这是一个 nil，直接返回。
		if original.IsNil() {
			return
		}
		// 获取接口指向的值，而不是指针。
		originalValue := original.Elem()

		// 获取值并调用 Elem()。
		copyValue := reflect.New(originalValue.Type()).Elem()
		copyRecursive(originalValue, copyValue)
		cpy.Set(copyValue)

	case reflect.Struct:
		t, ok := original.Interface().(time.Time)
		if ok {
			cpy.Set(reflect.ValueOf(t))
			return
		}
		// 遍历结构体的每个字段并复制它。
		for i := 0; i < original.NumField(); i++ {
			// 检查字段是否导出。
			// 结构体字段的 PkgPath 字段用于检查字段是否导出。
			// 如果 PkgPath 不为空，说明字段未导出。
			// 因为 CanSet() 返回 false 对于可设置的字段，所以需要检查 PkgPath。
			// -mohae
			if original.Type().Field(i).PkgPath != "" {
				continue
			}
			copyRecursive(original.Field(i), cpy.Field(i))
		}

	case reflect.Slice:
		if original.IsNil() {
			return
		}
		// 创建一个新的切片并复制每个元素。
		cpy.Set(reflect.MakeSlice(original.Type(), original.Len(), original.Cap()))
		for i := 0; i < original.Len(); i++ {
			copyRecursive(original.Index(i), cpy.Index(i))
		}

	case reflect.Map:
		if original.IsNil() {
			return
		}
		cpy.Set(reflect.MakeMap(original.Type()))
		for _, key := range original.MapKeys() {
			originalValue := original.MapIndex(key)
			copyValue := reflect.New(originalValue.Type()).Elem()
			copyRecursive(originalValue, copyValue)
			copyKey := Copy(key.Interface())
			cpy.SetMapIndex(reflect.ValueOf(copyKey), copyValue)
		}

	default:
		cpy.Set(original)
	}
}
