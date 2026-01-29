package utils

import "fmt"

// ListToMapByKey 将 `list` 转换为 map[string]interface{} 类型的数据结构，其中键由 `key` 指定。
// 注意，item的值可能是切片类型。
func ListToMapByKey(list []map[string]interface{}, key string) map[string]interface{} {
	var (
		s              = ""
		m              = make(map[string]interface{})
		tempMap        = make(map[string][]interface{})
		hasMultiValues bool
	)
	for _, item := range list {
		if k, ok := item[key]; ok {
			s = fmt.Sprintf(`%v`, k)
			tempMap[s] = append(tempMap[s], item)
			if len(tempMap[s]) > 1 {
				hasMultiValues = true
			}
		}
	}
	for k, v := range tempMap {
		if hasMultiValues {
			m[k] = v
		} else {
			m[k] = v[0]
		}
	}
	return m
}
