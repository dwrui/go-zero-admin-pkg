package gmap

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/empty"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gvar"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/json"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/rwmutex"
	"reflect"
)

// StrAnyMap 实现了带有RWMutex读写锁开关的 map[string]interface{}。
type StrAnyMap struct {
	mu   rwmutex.RWMutex        // 读写锁，用于并发安全控制
	data map[string]interface{} // 底层数据存储
}

// NewStrAnyMap 返回一个空的 StrAnyMap 对象。
// 参数 `safe` 用于指定是否使用并发安全模式，默认为 false。
func NewStrAnyMap(safe ...bool) *StrAnyMap {
	return &StrAnyMap{
		mu:   rwmutex.Create(safe...),
		data: make(map[string]interface{}),
	}
}

// NewStrAnyMapFrom 从给定的 map[string]interface{} 数据创建并返回一个哈希映射。
// 注意：参数 `data` 将被设置为底层数据映射（非深拷贝），
// 当在外部修改该映射时可能会存在并发安全问题。
func NewStrAnyMapFrom(data map[string]interface{}, safe ...bool) *StrAnyMap {
	return &StrAnyMap{
		mu:   rwmutex.Create(safe...),
		data: data,
	}
}

// Iterator 使用自定义回调函数 `f` 以只读方式迭代哈希映射。
// 如果 `f` 返回 true，则继续迭代；返回 false 则停止迭代。
func (m *StrAnyMap) Iterator(f func(k string, v interface{}) bool) {
	for k, v := range m.Map() {
		if !f(k, v) {
			break
		}
	}
}

// Clone 返回一个包含当前映射数据副本的新哈希映射。
func (m *StrAnyMap) Clone() *StrAnyMap {
	return NewStrAnyMapFrom(m.MapCopy(), m.mu.IsSafe())
}

// Map 返回底层数据映射。
// 注意：如果处于并发安全使用模式，它返回底层数据的副本，
// 否则返回指向底层数据的指针。
func (m *StrAnyMap) Map() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.mu.IsSafe() {
		return m.data
	}
	data := make(map[string]interface{}, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	return data
}

// MapStrAny 返回映射底层数据的副本，类型为 map[string]interface{}。
func (m *StrAnyMap) MapStrAny() map[string]interface{} {
	return m.Map()
}

// MapCopy 返回哈希映射底层数据的副本。
func (m *StrAnyMap) MapCopy() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data := make(map[string]interface{}, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	return data
}

// FilterEmpty 删除所有值为空的键值对。
// 以下值被视为空：0, nil, false, "", len(slice/map/chan) == 0。
func (m *StrAnyMap) FilterEmpty() {
	m.mu.Lock()
	for k, v := range m.data {
		if empty.IsEmpty(v) {
			delete(m.data, k)
		}
	}
	m.mu.Unlock()
}

// FilterNil 删除所有值为 nil 的键值对。
func (m *StrAnyMap) FilterNil() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range m.data {
		if empty.IsNil(v) {
			delete(m.data, k)
		}
	}
}

// Set 向哈希映射设置键值对。
func (m *StrAnyMap) Set(key string, val interface{}) {
	m.mu.Lock()
	if m.data == nil {
		m.data = make(map[string]interface{})
	}
	m.data[key] = val
	m.mu.Unlock()
}

// Sets 批量设置键值对到哈希映射。
func (m *StrAnyMap) Sets(data map[string]interface{}) {
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
// 第二个返回值 `found` 为 true 表示找到键，否则为 false。
func (m *StrAnyMap) Search(key string) (value interface{}, found bool) {
	m.mu.RLock()
	if m.data != nil {
		value, found = m.data[key]
	}
	m.mu.RUnlock()
	return
}

// Get 通过给定的 `key` 返回值。
func (m *StrAnyMap) Get(key string) (value interface{}) {
	m.mu.RLock()
	if m.data != nil {
		value = m.data[key]
	}
	m.mu.RUnlock()
	return
}

// Pop 从映射中检索并删除一个项目。
func (m *StrAnyMap) Pop() (key string, value interface{}) {
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
func (m *StrAnyMap) Pops(size int) map[string]interface{} {
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
		newMap = make(map[string]interface{}, size)
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

// doSetWithLockCheck 使用互斥锁检查键的值是否存在，
// 如果不存在，则使用给定的 `key` 将值设置到映射中，
// 否则只返回现有值。
//
// 当设置值时，如果 `value` 是 `func() interface{}` 类型，
// 它将在哈希映射的互斥锁保护下执行，
// 并将其返回值用 `key` 设置到映射中。
//
// 返回给定 `key` 的值。
func (m *StrAnyMap) doSetWithLockCheck(key string, value interface{}) interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string]interface{})
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

// GetOrSet 通过键返回值，如果该键不存在则使用给定的 `value` 设置值并返回该值。
func (m *StrAnyMap) GetOrSet(key string, value interface{}) interface{} {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, value)
	} else {
		return v
	}
}

// GetOrSetFunc 通过键返回值，如果该键不存在则使用回调函数 `f` 的返回值设置值并返回该值。
func (m *StrAnyMap) GetOrSetFunc(key string, f func() interface{}) interface{} {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, f())
	} else {
		return v
	}
}

// GetOrSetFuncLock 通过键返回值，如果该键不存在则使用回调函数 `f` 的返回值设置值并返回该值。
// GetOrSetFuncLock 与 GetOrSetFunc 的区别在于它在哈希映射的互斥锁保护下执行函数 `f`。
func (m *StrAnyMap) GetOrSetFuncLock(key string, f func() interface{}) interface{} {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, f)
	} else {
		return v
	}
}

// GetVar 返回给定 `key` 的值对应的 Var 对象。
// 返回的 Var 是非并发安全的。
func (m *StrAnyMap) GetVar(key string) *gvar.Var {
	return gvar.New(m.Get(key))
}

// GetVarOrSet 返回 GetVarOrSet 结果的 Var 对象。
// 返回的 Var 是非并发安全的。
func (m *StrAnyMap) GetVarOrSet(key string, value interface{}) *gvar.Var {
	return gvar.New(m.GetOrSet(key, value))
}

// GetVarOrSetFunc 返回 GetOrSetFunc 结果的 Var 对象。
// 返回的 Var 是非并发安全的。
func (m *StrAnyMap) GetVarOrSetFunc(key string, f func() interface{}) *gvar.Var {
	return gvar.New(m.GetOrSetFunc(key, f))
}

// GetVarOrSetFuncLock 返回 GetOrSetFuncLock 结果的 Var 对象。
// 返回的 Var 是非并发安全的。
func (m *StrAnyMap) GetVarOrSetFuncLock(key string, f func() interface{}) *gvar.Var {
	return gvar.New(m.GetOrSetFuncLock(key, f))
}

// SetIfNotExist 如果 `key` 不存在则将 `value` 设置到映射中，并返回 true。
// 如果 `key` 存在则返回 false，且 `value` 将被忽略。
func (m *StrAnyMap) SetIfNotExist(key string, value interface{}) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, value)
		return true
	}
	return false
}

// SetIfNotExistFunc 使用回调函数 `f` 的返回值设置值，并返回 true。
// 如果 `key` 存在则返回 false，且值将被忽略。
func (m *StrAnyMap) SetIfNotExistFunc(key string, f func() interface{}) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, f())
		return true
	}
	return false
}

// SetIfNotExistFuncLock 使用回调函数 `f` 的返回值设置值，并返回 true。
// 如果 `key` 存在则返回 false，且值将被忽略。
// SetIfNotExistFuncLock 与 SetIfNotExistFunc 的区别在于它在哈希映射的互斥锁保护下执行函数 `f`。
func (m *StrAnyMap) SetIfNotExistFuncLock(key string, f func() interface{}) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, f)
		return true
	}
	return false
}

// Removes 通过键批量删除映射中的值。
func (m *StrAnyMap) Removes(keys []string) {
	m.mu.Lock()
	if m.data != nil {
		for _, key := range keys {
			delete(m.data, key)
		}
	}
	m.mu.Unlock()
}

// Remove 通过给定的 `key` 从映射中删除值，并返回被删除的值。
func (m *StrAnyMap) Remove(key string) (value interface{}) {
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

// Keys 以切片形式返回映射的所有键。
func (m *StrAnyMap) Keys() []string {
	m.mu.RLock()
	var (
		keys  = make([]string, len(m.data))
		index = 0
	)
	for key := range m.data {
		keys[index] = key
		index++
	}
	m.mu.RUnlock()
	return keys
}

// Values 以切片形式返回映射的所有值。
func (m *StrAnyMap) Values() []interface{} {
	m.mu.RLock()
	var (
		values = make([]interface{}, len(m.data))
		index  = 0
	)
	for _, value := range m.data {
		values[index] = value
		index++
	}
	m.mu.RUnlock()
	return values
}

// Contains 检查键是否存在。
// 如果 `key` 存在返回 true，否则返回 false。
func (m *StrAnyMap) Contains(key string) bool {
	var ok bool
	m.mu.RLock()
	if m.data != nil {
		_, ok = m.data[key]
	}
	m.mu.RUnlock()
	return ok
}

// Size 返回映射的大小。
func (m *StrAnyMap) Size() int {
	m.mu.RLock()
	length := len(m.data)
	m.mu.RUnlock()
	return length
}

// IsEmpty 检查映射是否为空。
// 如果映射为空返回 true，否则返回 false。
func (m *StrAnyMap) IsEmpty() bool {
	return m.Size() == 0
}

// Clear 删除映射的所有数据，将重新创建一个新的底层数据映射。
func (m *StrAnyMap) Clear() {
	m.mu.Lock()
	m.data = make(map[string]interface{})
	m.mu.Unlock()
}

// Replace 用给定的 `data` 替换映射的数据。
func (m *StrAnyMap) Replace(data map[string]interface{}) {
	m.mu.Lock()
	m.data = data
	m.mu.Unlock()
}

// LockFunc 在 RWMutex.Lock 锁保护下使用给定的回调函数 `f` 锁定写入。
func (m *StrAnyMap) LockFunc(f func(m map[string]interface{})) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f(m.data)
}

// RLockFunc 在 RWMutex.RLock 锁保护下使用给定的回调函数 `f` 锁定读取。
func (m *StrAnyMap) RLockFunc(f func(m map[string]interface{})) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f(m.data)
}

// Flip 交换映射的键值对，将键值转换为值键。
func (m *StrAnyMap) Flip() {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := make(map[string]interface{}, len(m.data))
	for k, v := range m.data {
		n[gconv.String(v)] = k
	}
	m.data = n
}

// Merge 合并两个哈希映射。
// 参数 `other` 映射将被合并到映射 `m` 中。
func (m *StrAnyMap) Merge(other *StrAnyMap) {
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
func (m *StrAnyMap) String() string {
	if m == nil {
		return ""
	}
	b, _ := m.MarshalJSON()
	return string(b)
}

// MarshalJSON 实现 json.Marshal 的 MarshalJSON 接口。
func (m StrAnyMap) MarshalJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return json.Marshal(m.data)
}

// UnmarshalJSON 实现 json.Unmarshal 的 UnmarshalJSON 接口。
func (m *StrAnyMap) UnmarshalJSON(b []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string]interface{})
	}
	if err := json.UnmarshalUseNumber(b, &m.data); err != nil {
		return err
	}
	return nil
}

// UnmarshalValue 是一个接口实现，用于为映射设置任意类型的值。
func (m *StrAnyMap) UnmarshalValue(value interface{}) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = gconv.Map(value)
	return
}

// IsSubOf 检查当前映射是否是 `other` 映射的子映射。
func (m *StrAnyMap) IsSubOf(other *StrAnyMap) bool {
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

// Diff 比较当前映射 `m` 与映射 `other` 并返回它们的不同键。
// 返回的 `addedKeys` 是在映射 `m` 中但不在映射 `other` 中的键。
// 返回的 `removedKeys` 是在映射 `other` 中但不在映射 `m` 中的键。
// 返回的 `updatedKeys` 是同时在映射 `m` 和 `other` 中但值不相等 (`!=`) 的键。
func (m *StrAnyMap) Diff(other *StrAnyMap) (addedKeys, removedKeys, updatedKeys []string) {
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
