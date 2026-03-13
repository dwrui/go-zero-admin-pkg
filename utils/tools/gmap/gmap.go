// 包 gmap 提供了最常用的map容器，同时支持并发安全/不安全交换功能。
package gmap

type (
	Map     = AnyAnyMap // Map is alias of AnyAnyMap.
	HashMap = AnyAnyMap // HashMap is alias of AnyAnyMap.
)

// 新建生成并返回一个空哈希映射。
// 参数“安全”用于指定是否在并发安全中使用映射，
// 默认情况下是false。
func New(safe ...bool) *Map {
	return NewAnyAnyMap(safe...)
}

// 从给定的map `data`创建并返回一个哈希映射。
// 请注意，参数 `data` 映射将被设置为底层数据映射（无深拷贝），
// 因此在外部更改映射时可能会出现并发安全问题。
// 参数“安全”用于指定是否在并发安全中使用映射，
// 默认情况下是false。
func NewFrom(data map[interface{}]interface{}, safe ...bool) *Map {
	return NewAnyAnyMapFrom(data, safe...)
}

// NewHashMap creates and returns an empty hash map.
// 参数“安全”用于指定是否在并发安全中使用映射，
// 默认情况下是false。
func NewHashMap(safe ...bool) *Map {
	return NewAnyAnyMap(safe...)
}

// NewHashMapFrom 创建并返回一个从给定map `data`创建的哈希映射。
// 请注意，参数 `data` 映射将被设置为底层数据映射（无深拷贝），
// 因此在外部更改映射时可能会出现并发安全问题。
// 参数“安全”用于指定是否在并发安全中使用映射，
// 默认情况下是false。
func NewHashMapFrom(data map[interface{}]interface{}, safe ...bool) *Map {
	return NewAnyAnyMapFrom(data, safe...)
}
