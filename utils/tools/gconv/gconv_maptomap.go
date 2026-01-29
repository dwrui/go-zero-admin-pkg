package gconv

// MapToMap converts any map type variable `params` to another map type variable `pointer`
// using reflect.
// See doMapToMap.
func MapToMap(params any, pointer any, mapping ...map[string]string) error {
	return Scan(params, pointer, mapping...)
}
