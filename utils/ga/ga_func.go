package ga

import (
	"fmt"
	"reflect"

	"github.com/dwrui/go-zero-admin-pkg/utils/tools/gmd5"
	"github.com/dwrui/go-zero-admin-pkg/utils/tools/gvar"
)

// Md5 encryption
func Md5(str string) string {
	mdsecret, _ := gmd5.Encrypt(str)
	return mdsecret
}

// md5hex编码加密
func Md5Str(origin string) string {
	return gmd5.Md5StrHex(origin)
}

// 数据格式化成【】string
func FormatColumnData(data interface{}) []interface{} {
	if data == nil {
		return []interface{}{}
	}
	// 安全类型转换
	switch v := data.(type) {
	case []interface{}:
		column := make([]interface{}, len(v))
		for i, item := range v {
			// 处理每个元素的类型转换
			switch val := item.(type) {
			case string:
				column[i] = val
			case int, int8, int16, int32, int64:
				column[i] = fmt.Sprintf("%d", val)
			case float32, float64:
				column[i] = fmt.Sprintf("%v", val)
			default:
				column[i] = fmt.Sprintf("%v", val)
			}
		}
		return column
	case []string:
		column := make([]interface{}, len(v))
		for i, val := range v {
			column[i] = val
		}
		return column
	case []int:
		column := make([]interface{}, len(v))
		for i, val := range v {
			column[i] = fmt.Sprintf("%d", val)
		}
		return column
	default:
		// 对于其他类型，尝试转换为字符串数组
		return []interface{}{fmt.Sprintf("%v", data)}
	}
}

// ToInterfaceSlice 将任意类型转换为[]interface{}
// 支持[]string、[]int、[]int64、[]uint64、[]*gvar.Var等多种类型
func ToInterfaceSlice(data interface{}) []interface{} {
	if data == nil {
		return []interface{}{}
	}

	// 尝试直接类型断言
	switch v := data.(type) {
	case []interface{}:
		return v
	case []string:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	case []int:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	case []int64:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	case []uint64:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	case []*gvar.Var:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = val
		}
		return result
	default:
		// 使用反射处理其他类型
		return reflectToInterfaceSlice(data)
	}
}

// reflectToInterfaceSlice 使用反射将任意类型的切片转换为[]interface{}
func reflectToInterfaceSlice(data interface{}) []interface{} {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return []interface{}{data}
	}

	result := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = v.Index(i).Interface()
	}

	return result
}

/*
*
获取完整的URL
只返回后台的保存地址不增加
*/
func GetFullUrl(url string) string {
	if StrContains(url, "http://") || StrContains(url, "https://") {
		return url
	}
	if !StrContains(url, "common") {
		return "/common/" + url
	}
	return url
}
