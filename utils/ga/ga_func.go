package ga

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

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

// 非空 → 返回 "2006-01-02"
func FormatNullDate(nt sql.NullTime) string {
	if nt.Valid {
		return nt.Time.Format("2006-01-02")
	}
	return ""
}

// 如果你需要 datetime 格式：
func FormatTime(layout string, val any) string {
	switch v := val.(type) {
	case sql.NullTime:
		if v.Valid {
			return v.Time.Format(layout)
		}
		return v.Time.Format(layout)
	case int64:
		var t time.Time
		if v > 9999999999 {
			t = time.UnixMilli(v)
		} else {
			t = time.Unix(v, 0)
		}
		return t.Format(layout)
	case string:
		ts, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return ""
		}
		return FormatTime(layout, ts)
	}

	return ""
}

// 如果你需要 datetime 格式：
func FormatNullDateTime(nt sql.NullTime) string {
	if nt.Valid {
		return nt.Time.Format("2006-01-02 15:04")
	}
	return ""
}

// BuildTreeList 将扁平列表构建为树形结构
// list: 扁平数据列表
// rootParentId: 根节点的上级ID值（通常为0）
// parentField: 上级ID字段名（如 pid、base_unit_id）
// idField: 主键字段名（默认 id）
func BuildTreeList(list List, rootParentId int64, parentField string, idField ...string) List {
	idKey := "id"
	if len(idField) > 0 && idField[0] != "" {
		idKey = idField[0]
	}

	var result List
	for _, item := range list {
		if Int64(item[parentField]) != rootParentId {
			continue
		}
		node := item
		children := BuildTreeList(list, Int64(node[idKey]), parentField, idKey)
		if len(children) > 0 {
			node["children"] = children
		}
		result = append(result, node)
	}
	return result
}

// SpecNormalized 规格型号归一化，写入 spec_normalized 字段用于查重与检索；创建后不可修改。
// 规则：去首尾空白、全角转半角、小写、统一乘号/常用单位、去除标点与空白，保留字母数字与中文。
func SpecNormalized(spec string) string {
	s := strings.TrimSpace(spec)
	if s == "" {
		return ""
	}
	s = specFullWidthToHalfWidth(s)
	s = strings.ToLower(s)
	s = strings.NewReplacer(
		"×", "x", "＊", "x", "*", "x",
		"毫米", "mm", "㎜", "mm",
		"厘米", "cm", "公分", "cm",
		"千克", "kg", "公斤", "kg",
		"克", "g",
	).Replace(s)

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == 'x' {
			b.WriteRune(r)
			continue
		}
		if r >= 0x4e00 && r <= 0x9fff {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 200 {
		out = out[:200]
	}
	return out
}

func specFullWidthToHalfWidth(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == 0x3000:
			b.WriteRune(' ')
		case r >= 0xFF01 && r <= 0xFF5E:
			b.WriteRune(r - 0xFEE0)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
