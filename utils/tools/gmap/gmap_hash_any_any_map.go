package gmap

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/empty"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gvar"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/json"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/rwmutex"
	"reflect"
)

// AnyAnyMap 封装了 map[interface{}]interface{} 类型，并提供更多的 map 功能。
type AnyAnyMap struct {
	mu   rwmutex.RWMutex
	data map[interface{}]interface{}
}

// NewAnyAnyMap 创建并返回一个空的哈希映射。
// 参数 `safe` 用于指定是否使用并发安全，默认为 false。
func NewAnyAnyMap(safe ...bool) *AnyAnyMap {
	return &AnyAnyMap{
		mu:   rwmutex.Create(safe...),
		data: make(map[interface{}]interface{}),
	}
}

// NewAnyAnyMapFrom 从给定的 map `data` 创建并返回一个哈希映射。
// 注意，参数 `data` 映射将被设置为底层数据映射（不进行深拷贝），
// 当外部修改映射时可能会存在一些并发安全问题。
func NewAnyAnyMapFrom(data map[interface{}]interface{}, safe ...bool) *AnyAnyMap {
	return &AnyAnyMap{
		mu:   rwmutex.Create(safe...),
		data: data,
	}
}

// Iterator 使用自定义回调函数 `f` 以只读方式迭代哈希映射。
// 如果 `f` 返回 true，则继续迭代；返回 false 则停止。
func (m *AnyAnyMap) Iterator(f func(k interface{}, v interface{}) bool) {
	for k, v := range m.Map() {
		if !f(k, v) {
			break
		}
	}
}

// Clone 返回一个包含当前映射数据副本的新哈希映射。
func (m *AnyAnyMap) Clone(safe ...bool) *AnyAnyMap {
	return NewFrom(m.MapCopy(), safe...)
}

// Map 返回底层数据映射。
// 注意，如果处于并发安全使用状态，它返回底层数据的副本，
// 否则返回指向底层数据的指针。
func (m *AnyAnyMap) Map() map[interface{}]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.mu.IsSafe() {
		return m.data
	}
	data := make(map[interface{}]interface{}, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	return data
}

// MapCopy 返回哈希映射底层数据的浅拷贝。
func (m *AnyAnyMap) MapCopy() map[interface{}]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data := make(map[interface{}]interface{}, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	return data
}

// MapStrAny 以 map[string]interface{} 的形式返回映射底层数据的副本。
func (m *AnyAnyMap) MapStrAny() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data := make(map[string]interface{}, len(m.data))
	for k, v := range m.data {
		data[gconv.String(k)] = v
	}
	return data
}

// FilterEmpty 删除所有值为空的键值对。
// 以下值被视为空：0、nil、false、""、len(slice/map/chan) == 0。
func (m *AnyAnyMap) FilterEmpty() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range m.data {
		if empty.IsEmpty(v) {
			delete(m.data, k)
		}
	}
}

// FilterNil 删除所有值为 nil 的键值对。
func (m *AnyAnyMap) FilterNil() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range m.data {
		if empty.IsNil(v) {
			delete(m.data, k)
		}
	}
}

// Set 向哈希映射设置键值。
func (m *AnyAnyMap) Set(key interface{}, value interface{}) {
	m.mu.Lock()
	if m.data == nil {
		m.data = make(map[interface{}]interface{})
	}
	m.data[key] = value
	m.mu.Unlock()
}

// Sets 批量设置键值到哈希映射。
func (m *AnyAnyMap) Sets(data map[interface{}]interface{}) {
	m.mu.Lock()
	if m.data == nil {
		m.data = data
	} else {
		for k, v := range data {
			m.data[k] = v
		}
	}
	m.mu.Unlock()
}

// Search 使用给定的 `key` 搜索映射。
// 第二个返回值 `found` 为 true 表示找到了键，否则为 false。
func (m *AnyAnyMap) Search(key interface{}) (value interface{}, found bool) {
	m.mu.RLock()
	if m.data != nil {
		value, found = m.data[key]
	}
	m.mu.RUnlock()
	return
}

// Get 通过给定的 `key` 返回值。
func (m *AnyAnyMap) Get(key interface{}) (value interface{}) {
	m.mu.RLock()
	if m.data != nil {
		value = m.data[key]
	}
	m.mu.RUnlock()
	return
}

// Pop 从映射中检索并删除一个项目。
func (m *AnyAnyMap) Pop() (key, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, value = range m.data {
		delete(m.data, key)
		return
	}
	return
}

// Pops 从映射中检索并删除 `size` 个项目。
// 如果 size == -1，则返回所有项目。
func (m *AnyAnyMap) Pops(size int) map[interface{}]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if size > len(m.data) || size == -1 {
		size = len(m.data)
	}
	if size == 0 {
		return nil
	}
	var (
		index  = 0
		newMap = make(map[interface{}]interface{}, size)
	)
	for k, v := range m.data {
		delete(m.data, k)
		newMap[k] = v
		index++
		if index == size {
			break
		}
	}
	return newMap
}

// doSetWithLockCheck 使用 mutex.Lock 检查键的值是否存在，
// 如果不存在，则使用给定的 `key` 将值设置到映射中，
// 否则只返回现有值。
//
// 设置值时，如果 `value` 是 `func() interface {}` 类型，
// 它将在哈希映射的 mutex.Lock 保护下执行，
// 其返回值将使用 `key` 设置到映射中。
//
// 它返回给定 `key` 的值。
func (m *AnyAnyMap) doSetWithLockCheck(key interface{}, value interface{}) interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[interface{}]interface{})
	}
	if v, ok := m.data[key]; ok {
		return v
	}
	if f, ok := value.(func() interface{}); ok {
		value = f()
	}
	if value != nil {
		m.data[key] = value
	}
	return value
}

// GetOrSet 通过键返回值，
// 如果不存在则使用给定的 `value` 设置值并返回该值。
func (m *AnyAnyMap) GetOrSet(key interface{}, value interface{}) interface{} {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, value)
	} else {
		return v
	}
}

// GetOrSetFunc 通过键返回值，
// 如果不存在则使用回调函数 `f` 的返回值设置值并返回该值。
func (m *AnyAnyMap) GetOrSetFunc(key interface{}, f func() interface{}) interface{} {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, f())
	} else {
		return v
	}
}

// GetOrSetFuncLock 通过键返回值，
// 如果不存在则使用回调函数 `f` 的返回值设置值并返回该值。
//
// GetOrSetFuncLock 与 GetOrSetFunc 函数的区别在于它使用哈希映射的 mutex.Lock 执行函数 `f`。
func (m *AnyAnyMap) GetOrSetFuncLock(key interface{}, f func() interface{}) interface{} {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, f)
	} else {
		return v
	}
}

// GetVar 返回一个包含给定 `key` 值的 Var。
// 返回的 Var 是非并发安全的。
func (m *AnyAnyMap) GetVar(key interface{}) *gvar.Var {
	return gvar.New(m.Get(key))
}

// GetVarOrSet 返回一个来自 GetOrSet 结果的 Var。
// 返回的 Var 是非并发安全的。
func (m *AnyAnyMap) GetVarOrSet(key interface{}, value interface{}) *gvar.Var {
	return gvar.New(m.GetOrSet(key, value))
}

// GetVarOrSetFunc 返回一个来自 GetOrSetFunc 结果的 Var。
// 返回的 Var 是非并发安全的。
func (m *AnyAnyMap) GetVarOrSetFunc(key interface{}, f func() interface{}) *gvar.Var {
	return gvar.New(m.GetOrSetFunc(key, f))
}

// GetVarOrSetFuncLock 返回一个来自 GetOrSetFuncLock 结果的 Var。
// 返回的 Var 是非并发安全的。
func (m *AnyAnyMap) GetVarOrSetFuncLock(key interface{}, f func() interface{}) *gvar.Var {
	return gvar.New(m.GetOrSetFuncLock(key, f))
}

// SetIfNotExist 如果 `key` 不存在则将 `value` 设置到映射中，然后返回 true。
// 如果 `key` 存在则返回 false，`value` 将被忽略。
func (m *AnyAnyMap) SetIfNotExist(key interface{}, value interface{}) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, value)
		return true
	}
	return false
}

// SetIfNotExistFunc 使用回调函数 `f` 的返回值设置值，然后返回 true。
// 如果 `key` 存在则返回 false，`value` 将被忽略。
func (m *AnyAnyMap) SetIfNotExistFunc(key interface{}, f func() interface{}) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, f())
		return true
	}
	return false
}

// SetIfNotExistFuncLock 使用回调函数 `f` 的返回值设置值，然后返回 true。
// 如果 `key` 存在则返回 false，`value` 将被忽略。
//
// SetIfNotExistFuncLock 与 SetIfNotExistFunc 函数的区别在于
// 它使用哈希映射的 mutex.Lock 执行函数 `f`。
func (m *AnyAnyMap) SetIfNotExistFuncLock(key interface{}, f func() interface{}) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, f)
		return true
	}
	return false
}

// Remove 通过给定的 `key` 从映射中删除值，并返回删除的值。
func (m *AnyAnyMap) Remove(key interface{}) (value interface{}) {
	m.mu.Lock()
	if m.data != nil {
		var ok bool
		if value, ok = m.data[key]; ok {
			delete(m.data, key)
		}
	}
	m.mu.Unlock()
	return
}

// Removes 批量通过键删除映射中的值。
func (m *AnyAnyMap) Removes(keys []interface{}) {
	m.mu.Lock()
	if m.data != nil {
		for _, key := range keys {
			delete(m.data, key)
		}
	}
	m.mu.Unlock()
}

// Keys 以切片形式返回映射的所有键。
func (m *AnyAnyMap) Keys() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var (
		keys  = make([]interface{}, len(m.data))
		index = 0
	)
	for key := range m.data {
		keys[index] = key
		index++
	}
	return keys
}

// Values 以切片形式返回映射的所有值。
func (m *AnyAnyMap) Values() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var (
		values = make([]interface{}, len(m.data))
		index  = 0
	)
	for _, value := range m.data {
		values[index] = value
		index++
	}
	return values
}

// Contains 检查键是否存在。
// 如果 `key` 存在则返回 true，否则返回 false。
func (m *AnyAnyMap) Contains(key interface{}) bool {
	var ok bool
	m.mu.RLock()
	if m.data != nil {
		_, ok = m.data[key]
	}
	m.mu.RUnlock()
	return ok
}

// Size 返回映射的大小。
func (m *AnyAnyMap) Size() int {
	m.mu.RLock()
	length := len(m.data)
	m.mu.RUnlock()
	return length
}

// IsEmpty 检查映射是否为空。
// 如果映射为空则返回 true，否则返回 false。
func (m *AnyAnyMap) IsEmpty() bool {
	return m.Size() == 0
}

// Clear 删除映射的所有数据，它将重新创建一个新的底层数据映射。
func (m *AnyAnyMap) Clear() {
	m.mu.Lock()
	m.data = make(map[interface{}]interface{})
	m.mu.Unlock()
}

// Replace 用给定的 `data` 替换映射的数据。
func (m *AnyAnyMap) Replace(data map[interface{}]interface{}) {
	m.mu.Lock()
	m.data = data
	m.mu.Unlock()
}

// LockFunc 使用 RWMutex.Lock 锁定写入，并在锁定期间执行给定的回调函数 `f`。
func (m *AnyAnyMap) LockFunc(f func(m map[interface{}]interface{})) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f(m.data)
}

// RLockFunc 使用 RWMutex.RLock 锁定读取，并在锁定期间执行给定的回调函数 `f`。
func (m *AnyAnyMap) RLockFunc(f func(m map[interface{}]interface{})) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f(m.data)
}

// Flip 交换映射的键值对，将键变为值，值变为键。
func (m *AnyAnyMap) Flip() {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := make(map[interface{}]interface{}, len(m.data))
	for k, v := range m.data {
		n[v] = k
	}
	m.data = n
}

// Merge 合并两个哈希映射。
// `other` 映射将被合并到映射 `m` 中。
func (m *AnyAnyMap) Merge(other *AnyAnyMap) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = other.MapCopy()
		return
	}
	if other != m {
		other.mu.RLock()
		defer other.mu.RUnlock()
	}
	for k, v := range other.data {
		m.data[k] = v
	}
}

// String 将映射作为字符串返回。
func (m *AnyAnyMap) String() string {
	if m == nil {
		return ""
	}
	b, _ := m.MarshalJSON()
	return string(b)
}

// MarshalJSON 实现 json.Marshal 的 MarshalJSON 接口。
func (m AnyAnyMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(gconv.Map(m.Map()))
}

// UnmarshalJSON 实现 json.Unmarshal 的 UnmarshalJSON 接口。
func (m *AnyAnyMap) UnmarshalJSON(b []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[interface{}]interface{})
	}
	var data map[string]interface{}
	if err := json.UnmarshalUseNumber(b, &data); err != nil {
		return err
	}
	for k, v := range data {
		m.data[k] = v
	}
	return nil
}

// UnmarshalValue 是一个接口实现，用于为映射设置任何类型的值。
func (m *AnyAnyMap) UnmarshalValue(value interface{}) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[interface{}]interface{})
	}
	for k, v := range gconv.Map(value) {
		m.data[k] = v
	}
	return
}

// IsSubOf 检查当前映射是否是 `other` 映射的子映射。
func (m *AnyAnyMap) IsSubOf(other *AnyAnyMap) bool {
	if m == other {
		return true
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()
	for key, value := range m.data {
		otherValue, ok := other.data[key]
		if !ok {
			return false
		}
		if otherValue != value {
			return false
		}
	}
	return true
}

// Diff 比较映射 `m` 与映射 `other` 并返回它们的不同键。
// 返回的 `addedKeys` 是在映射 `m` 中但不在映射 `other` 中的键。
// 返回的 `removedKeys` 是在映射 `other` 中但不在映射 `m` 中的键。
// 返回的 `updatedKeys` 是同时在映射 `m` 和 `other` 中但它们的值不相等（!=）的键。
func (m *AnyAnyMap) Diff(other *AnyAnyMap) (addedKeys, removedKeys, updatedKeys []interface{}) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()

	for key := range m.data {
		if _, ok := other.data[key]; !ok {
			removedKeys = append(removedKeys, key)
		} else if !reflect.DeepEqual(m.data[key], other.data[key]) {
			updatedKeys = append(updatedKeys, key)
		}
	}
	for key := range other.data {
		if _, ok := m.data[key]; !ok {
			addedKeys = append(addedKeys, key)
		}
	}
	return
}
