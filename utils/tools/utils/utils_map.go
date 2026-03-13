package utils

// MapPossibleItemByKey 尝试为给定的键查找可能的键值对，忽略大小写和符号。
//
// 注意，此函数性能可能较低。
func MapPossibleItemByKey(data map[string]interface{}, key string) (foundKey string, foundValue interface{}) {
	if len(data) == 0 {
		return
	}
	if v, ok := data[key]; ok {
		return key, v
	}
	// Loop checking.
	for k, v := range data {
		if EqualFoldWithoutChars(k, key) {
			return k, v
		}
	}
	return "", nil
}

// MapContainsPossibleKey checks if the given `key` is contained in given map `data`.
// It checks the key ignoring cases and symbols.
//
// Note that this function might be of low performance.
func MapContainsPossibleKey(data map[string]interface{}, key string) bool {
	if k, _ := MapPossibleItemByKey(data, key); k != "" {
		return true
	}
	return false
}
