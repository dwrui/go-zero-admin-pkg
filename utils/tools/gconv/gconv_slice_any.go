package gconv

// SliceAny is alias of Interfaces.
func SliceAny(any interface{}) []interface{} {
	return Interfaces(any)
}

// Interfaces converts `any` to []interface{}.
func Interfaces(any interface{}) []interface{} {
	result, _ := defaultConverter.SliceAny(any, SliceOption{
		ContinueOnError: true,
	})
	return result
}
