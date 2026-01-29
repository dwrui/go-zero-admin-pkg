package gmap

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/empty"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/json"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/rwmutex"
)

// IntStrMap 实现了带有 RWMutex 开关的 map[int]string。
type IntStrMap struct {
	mu   rwmutex.RWMutex
	data map[int]string
}

// NewIntStrMap 返回一个空的 IntStrMap 对象。
// 参数 `safe` 用于指定是否使用并发安全，默认为 false。
func NewIntStrMap(safe ...bool) *IntStrMap {
	return &IntStrMap{
		mu:   rwmutex.Create(safe...),
		data: make(map[int]string),
	}
}

// NewIntStrMapFrom 从给定的 map `data` 创建并返回一个哈希映射。
// 注意，参数 `data` 映射将被设置为底层数据映射（不进行深拷贝），
// 当外部修改映射时可能会存在一些并发安全问题。
func NewIntStrMapFrom(data map[int]string, safe ...bool) *IntStrMap {
	return &IntStrMap{
		mu:   rwmutex.Create(safe...),
		data: data,
	}
}

// Iterator 使用自定义回调函数 `f` 以只读方式迭代哈希映射。
// 如果 `f` 返回 true，则继续迭代；返回 false 则停止。
func (m *IntStrMap) Iterator(f func(k int, v string) bool) {
	for k, v := range m.Map() {
		if !f(k, v) {
			break
		}
	}
}

// Clone 返回一个包含当前映射数据副本的新哈希映射。
func (m *IntStrMap) Clone() *IntStrMap {
	return NewIntStrMapFrom(m.MapCopy(), m.mu.IsSafe())
}

// Map 返回底层数据映射。
// 注意，如果处于并发安全使用状态，它返回底层数据的副本，
// 否则返回指向底层数据的指针。
func (m *IntStrMap) Map() map[int]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.mu.IsSafe() {
		return m.data
	}
	data := make(map[int]string, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	return data
}

// MapStrAny 以 map[string]interface{} 的形式返回映射底层数据的副本。
func (m *IntStrMap) MapStrAny() map[string]interface{} {
	m.mu.RLock()
	data := make(map[string]interface{}, len(m.data))
	for k, v := range m.data {
		data[gconv.String(k)] = v
	}
	m.mu.RUnlock()
	return data
}

// MapCopy 返回哈希映射底层数据的副本。
func (m *IntStrMap) MapCopy() map[int]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data := make(map[int]string, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	return data
}

// FilterEmpty 删除所有值为空的键值对。
// 以下值被视为空：0、nil、false、""、len(slice/map/chan) == 0。
func (m *IntStrMap) FilterEmpty() {
	m.mu.Lock()
	for k, v := range m.data {
		if empty.IsEmpty(v) {
			delete(m.data, k)
		}
	}
	m.mu.Unlock()
}

// Set 向哈希映射设置键值。
func (m *IntStrMap) Set(key int, val string) {
	m.mu.Lock()
	if m.data == nil {
		m.data = make(map[int]string)
	}
	m.data[key] = val
	m.mu.Unlock()
}

// Sets 批量设置键值到哈希映射。
func (m *IntStrMap) Sets(data map[int]string) {
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
func (m *IntStrMap) Search(key int) (value string, found bool) {
	m.mu.RLock()
	if m.data != nil {
		value, found = m.data[key]
	}
	m.mu.RUnlock()
	return
}

// Get 通过给定的 `key` 返回值。
func (m *IntStrMap) Get(key int) (value string) {
	m.mu.RLock()
	if m.data != nil {
		value = m.data[key]
	}
	m.mu.RUnlock()
	return
}

// Pop 从映射中检索并删除一个项目。
func (m *IntStrMap) Pop() (key int, value string) {
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
func (m *IntStrMap) Pops(size int) map[int]string {
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
		newMap = make(map[int]string, size)
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
// 它返回给定 `key` 的值。
func (m *IntStrMap) doSetWithLockCheck(key int, value string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[int]string)
	}
	if v, ok := m.data[key]; ok {
		return v
	}
	m.data[key] = value
	return value
}

// GetOrSet 通过键返回值，
// 如果不存在则使用给定的 `value` 设置值并返回该值。
func (m *IntStrMap) GetOrSet(key int, value string) string {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, value)
	} else {
		return v
	}
}

// GetOrSetFunc 通过键返回值，
// 如果不存在则使用回调函数 `f` 的返回值设置值并返回该值。
func (m *IntStrMap) GetOrSetFunc(key int, f func() string) string {
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
func (m *IntStrMap) GetOrSetFuncLock(key int, f func() string) string {
	if v, ok := m.Search(key); !ok {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.data == nil {
			m.data = make(map[int]string)
		}
		if v, ok = m.data[key]; ok {
			return v
		}
		v = f()
		m.data[key] = v
		return v
	} else {
		return v
	}
}

// SetIfNotExist 如果 `key` 不存在则将 `value` 设置到映射中，然后返回 true。
// 如果 `key` 存在则返回 false，`value` 将被忽略。
func (m *IntStrMap) SetIfNotExist(key int, value string) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, value)
		return true
	}
	return false
}

// SetIfNotExistFunc 使用回调函数 `f` 的返回值设置值，然后返回 true。
// 如果 `key` 存在则返回 false，`value` 将被忽略。
func (m *IntStrMap) SetIfNotExistFunc(key int, f func() string) bool {
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
func (m *IntStrMap) SetIfNotExistFuncLock(key int, f func() string) bool {
	if !m.Contains(key) {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.data == nil {
			m.data = make(map[int]string)
		}
		if _, ok := m.data[key]; !ok {
			m.data[key] = f()
		}
		return true
	}
	return false
}

// Removes 批量通过键删除映射中的值。
func (m *IntStrMap) Removes(keys []int) {
	m.mu.Lock()
	if m.data != nil {
		for _, key := range keys {
			delete(m.data, key)
		}
	}
	m.mu.Unlock()
}

// Remove 通过给定的 `key` 从映射中删除值，并返回删除的值。
func (m *IntStrMap) Remove(key int) (value string) {
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
func (m *IntStrMap) Keys() []int {
	m.mu.RLock()
	var (
		keys  = make([]int, len(m.data))
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
func (m *IntStrMap) Values() []string {
	m.mu.RLock()
	var (
		values = make([]string, len(m.data))
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
// 如果 `key` 存在则返回 true，否则返回 false。
func (m *IntStrMap) Contains(key int) bool {
	var ok bool
	m.mu.RLock()
	if m.data != nil {
		_, ok = m.data[key]
	}
	m.mu.RUnlock()
	return ok
}

// Size 返回映射的大小。
func (m *IntStrMap) Size() int {
	m.mu.RLock()
	length := len(m.data)
	m.mu.RUnlock()
	return length
}

// IsEmpty 检查映射是否为空。
// 如果映射为空则返回 true，否则返回 false。
func (m *IntStrMap) IsEmpty() bool {
	return m.Size() == 0
}

// Clear 删除映射的所有数据，它将重新创建一个新的底层数据映射。
func (m *IntStrMap) Clear() {
	m.mu.Lock()
	m.data = make(map[int]string)
	m.mu.Unlock()
}

// Replace 用给定的 `data` 替换映射的数据。
func (m *IntStrMap) Replace(data map[int]string) {
	m.mu.Lock()
	m.data = data
	m.mu.Unlock()
}

// LockFunc 使用 RWMutex.Lock 锁定写入，并在锁定期间执行给定的回调函数 `f`。
func (m *IntStrMap) LockFunc(f func(m map[int]string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f(m.data)
}

// RLockFunc 使用 RWMutex.RLock 锁定读取，并在锁定期间执行给定的回调函数 `f`。
func (m *IntStrMap) RLockFunc(f func(m map[int]string)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f(m.data)
}

// Flip 交换映射的键值对，将键变为值，值变为键。
// 注意：由于值是字符串类型，需要转换为整数作为新键。
func (m *IntStrMap) Flip() {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := make(map[int]string, len(m.data))
	for k, v := range m.data {
		n[gconv.Int(v)] = gconv.String(k)
	}
	m.data = n
}

// Merge 合并两个哈希映射。
// `other` 映射将被合并到映射 `m` 中。
func (m *IntStrMap) Merge(other *IntStrMap) {
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
func (m *IntStrMap) String() string {
	if m == nil {
		return ""
	}
	b, _ := m.MarshalJSON()
	return string(b)
}

// MarshalJSON 实现 json.Marshal 的 MarshalJSON 接口。
func (m IntStrMap) MarshalJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return json.Marshal(m.data)
}

// UnmarshalJSON 实现 json.Unmarshal 的 UnmarshalJSON 接口。
func (m *IntStrMap) UnmarshalJSON(b []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[int]string)
	}
	if err := json.UnmarshalUseNumber(b, &m.data); err != nil {
		return err
	}
	return nil
}

// UnmarshalValue 是一个接口实现，用于为映射设置任何类型的值。
func (m *IntStrMap) UnmarshalValue(value interface{}) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[int]string)
	}
	switch value.(type) {
	case string, []byte:
		return json.UnmarshalUseNumber(gconv.Bytes(value), &m.data)
	default:
		for k, v := range gconv.Map(value) {
			m.data[gconv.Int(k)] = gconv.String(v)
		}
	}
	return
}

// DeepCopy 实现当前类型的深拷贝接口。
func (m *IntStrMap) DeepCopy() interface{} {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	data := make(map[int]string, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	return NewIntStrMapFrom(data, m.mu.IsSafe())
}

// IsSubOf 检查当前映射是否是 `other` 映射的子映射。
func (m *IntStrMap) IsSubOf(other *IntStrMap) bool {
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
// 返回的 `updatedKeys` 是同时在映射 `m` 和 `other` 中但它们的值不相等的键。
func (m *IntStrMap) Diff(other *IntStrMap) (addedKeys, removedKeys, updatedKeys []int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()

	for key := range m.data {
		if _, ok := other.data[key]; !ok {
			removedKeys = append(removedKeys, key)
		} else if m.data[key] != other.data[key] {
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
