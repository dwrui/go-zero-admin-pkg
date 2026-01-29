package gconv

// MapToMaps converts any slice type variable `params` to another map slice type variable `pointer`.
// See doMapToMaps.
func MapToMaps(params any, pointer any, mapping ...map[string]string) error {
	return Scan(params, pointer, mapping...)
}
