package gmap

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/empty"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/json"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/rwmutex"
)

// StrStrMap 实现了带有RWMutex读写锁开关的 map[string]string。
type StrStrMap struct {
	mu   rwmutex.RWMutex   // 读写锁，用于并发安全控制
	data map[string]string // 底层数据存储
}

// NewStrStrMap 返回一个空的 StrStrMap 对象。
// 参数 `safe` 用于指定是否使用并发安全模式，默认为 false。
func NewStrStrMap(safe ...bool) *StrStrMap {
	return &StrStrMap{
		data: make(map[string]string),
		mu:   rwmutex.Create(safe...),
	}
}

// NewStrStrMapFrom 从给定的 map[string]string 数据创建并返回一个哈希映射。
// 注意：参数 `data` 将被设置为底层数据映射（非深拷贝），
// 当在外部修改该映射时可能会存在并发安全问题。
func NewStrStrMapFrom(data map[string]string, safe ...bool) *StrStrMap {
	return &StrStrMap{
		mu:   rwmutex.Create(safe...),
		data: data,
	}
}

// Iterator 使用自定义回调函数 `f` 以只读方式迭代哈希映射。
// 如果 `f` 返回 true，则继续迭代；返回 false 则停止迭代。
func (m *StrStrMap) Iterator(f func(k string, v string) bool) {
	for k, v := range m.Map() {
		if !f(k, v) {
			break
		}
	}
}

// Clone 返回一个包含当前映射数据副本的新哈希映射。
func (m *StrStrMap) Clone() *StrStrMap {
	return NewStrStrMapFrom(m.MapCopy(), m.mu.IsSafe())
}

// Map 返回底层数据映射。
// 注意：如果处于并发安全使用模式，它返回底层数据的副本，
// 否则返回指向底层数据的指针。
func (m *StrStrMap) Map() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.mu.IsSafe() {
		return m.data
	}
	data := make(map[string]string, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	return data
}

// MapStrAny 返回映射底层数据的副本，类型为 map[string]interface{}。
func (m *StrStrMap) MapStrAny() map[string]interface{} {
	m.mu.RLock()
	data := make(map[string]interface{}, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	m.mu.RUnlock()
	return data
}

// MapCopy 返回哈希映射底层数据的副本。
func (m *StrStrMap) MapCopy() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data := make(map[string]string, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	return data
}

// FilterEmpty 删除所有值为空的键值对。
// 以下值被视为空：0, nil, false, "", len(slice/map/chan) == 0。
func (m *StrStrMap) FilterEmpty() {
	m.mu.Lock()
	for k, v := range m.data {
		if empty.IsEmpty(v) {
			delete(m.data, k)
		}
	}
	m.mu.Unlock()
}

// Set 向哈希映射设置键值对。
func (m *StrStrMap) Set(key string, val string) {
	m.mu.Lock()
	if m.data == nil {
		m.data = make(map[string]string)
	}
	m.data[key] = val
	m.mu.Unlock()
}

// Sets 批量设置键值对到哈希映射。
func (m *StrStrMap) Sets(data map[string]string) {
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
func (m *StrStrMap) Search(key string) (value string, found bool) {
	m.mu.RLock()
	if m.data != nil {
		value, found = m.data[key]
	}
	m.mu.RUnlock()
	return
}

// Get 通过给定的 `key` 返回值。
func (m *StrStrMap) Get(key string) (value string) {
	m.mu.RLock()
	if m.data != nil {
		value = m.data[key]
	}
	m.mu.RUnlock()
	return
}

// Pop 从映射中检索并删除一个项目。
func (m *StrStrMap) Pop() (key, value string) {
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
func (m *StrStrMap) Pops(size int) map[string]string {
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
		newMap = make(map[string]string, size)
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
// 返回给定 `key` 的值。
func (m *StrStrMap) doSetWithLockCheck(key string, value string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string]string)
	}
	if v, ok := m.data[key]; ok {
		return v
	}
	m.data[key] = value
	return value
}

// GetOrSet 通过键返回值，如果该键不存在则使用给定的 `value` 设置值并返回该值。
func (m *StrStrMap) GetOrSet(key string, value string) string {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, value)
	} else {
		return v
	}
}

// GetOrSetFunc 通过键返回值，如果该键不存在则使用回调函数 `f` 的返回值设置值并返回该值。
func (m *StrStrMap) GetOrSetFunc(key string, f func() string) string {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, f())
	} else {
		return v
	}
}

// GetOrSetFuncLock 通过键返回值，如果该键不存在则使用回调函数 `f` 的返回值设置值并返回该值。
// GetOrSetFuncLock 与 GetOrSetFunc 的区别在于它在哈希映射的互斥锁保护下执行函数 `f`。
func (m *StrStrMap) GetOrSetFuncLock(key string, f func() string) string {
	if v, ok := m.Search(key); !ok {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.data == nil {
			m.data = make(map[string]string)
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

// SetIfNotExist 如果 `key` 不存在则将 `value` 设置到映射中，并返回 true。
// 如果 `key` 存在则返回 false，且 `value` 将被忽略。
func (m *StrStrMap) SetIfNotExist(key string, value string) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, value)
		return true
	}
	return false
}

// SetIfNotExistFunc 使用回调函数 `f` 的返回值设置值，并返回 true。
// 如果 `key` 存在则返回 false，且值将被忽略。
func (m *StrStrMap) SetIfNotExistFunc(key string, f func() string) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, f())
		return true
	}
	return false
}

// SetIfNotExistFuncLock 使用回调函数 `f` 的返回值设置值，并返回 true。
// 如果 `key` 存在则返回 false，且值将被忽略。
// SetIfNotExistFuncLock 与 SetIfNotExistFunc 的区别在于它在哈希映射的互斥锁保护下执行函数 `f`。
func (m *StrStrMap) SetIfNotExistFuncLock(key string, f func() string) bool {
	if !m.Contains(key) {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.data == nil {
			m.data = make(map[string]string)
		}
		if _, ok := m.data[key]; !ok {
			m.data[key] = f()
		}
		return true
	}
	return false
}

// Removes 通过键批量删除映射中的值。
func (m *StrStrMap) Removes(keys []string) {
	m.mu.Lock()
	if m.data != nil {
		for _, key := range keys {
			delete(m.data, key)
		}
	}
	m.mu.Unlock()
}

// Remove 通过给定的 `key` 从映射中删除值，并返回被删除的值。
func (m *StrStrMap) Remove(key string) (value string) {
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
func (m *StrStrMap) Keys() []string {
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
func (m *StrStrMap) Values() []string {
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
// 如果 `key` 存在返回 true，否则返回 false。
func (m *StrStrMap) Contains(key string) bool {
	var ok bool
	m.mu.RLock()
	if m.data != nil {
		_, ok = m.data[key]
	}
	m.mu.RUnlock()
	return ok
}

// Size 返回映射的大小。
func (m *StrStrMap) Size() int {
	m.mu.RLock()
	length := len(m.data)
	m.mu.RUnlock()
	return length
}

// IsEmpty 检查映射是否为空。
// 如果映射为空返回 true，否则返回 false。
func (m *StrStrMap) IsEmpty() bool {
	return m.Size() == 0
}

// Clear 删除映射的所有数据，将重新创建一个新的底层数据映射。
func (m *StrStrMap) Clear() {
	m.mu.Lock()
	m.data = make(map[string]string)
	m.mu.Unlock()
}

// Replace 用给定的 `data` 替换映射的数据。
func (m *StrStrMap) Replace(data map[string]string) {
	m.mu.Lock()
	m.data = data
	m.mu.Unlock()
}

// LockFunc 在 RWMutex.Lock 锁保护下使用给定的回调函数 `f` 锁定写入。
func (m *StrStrMap) LockFunc(f func(m map[string]string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f(m.data)
}

// RLockFunc 在 RWMutex.RLock 锁保护下使用给定的回调函数 `f` 锁定读取。
func (m *StrStrMap) RLockFunc(f func(m map[string]string)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f(m.data)
}

// Flip 交换映射的键值对，将键值转换为值键。
func (m *StrStrMap) Flip() {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := make(map[string]string, len(m.data))
	for k, v := range m.data {
		n[v] = k
	}
	m.data = n
}

// Merge 合并两个哈希映射。
// 参数 `other` 映射将被合并到映射 `m` 中。
func (m *StrStrMap) Merge(other *StrStrMap) {
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
func (m *StrStrMap) String() string {
	if m == nil {
		return ""
	}
	b, _ := m.MarshalJSON()
	return string(b)
}

// MarshalJSON 实现 json.Marshal 的 MarshalJSON 接口。
func (m StrStrMap) MarshalJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return json.Marshal(m.data)
}

// UnmarshalJSON 实现 json.Unmarshal 的 UnmarshalJSON 接口。
func (m *StrStrMap) UnmarshalJSON(b []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string]string)
	}
	if err := json.UnmarshalUseNumber(b, &m.data); err != nil {
		return err
	}
	return nil
}

// UnmarshalValue 是一个接口实现，用于为映射设置任意类型的值。
func (m *StrStrMap) UnmarshalValue(value interface{}) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = gconv.MapStrStr(value)
	return
}

// DeepCopy 实现当前类型的深拷贝接口。
func (m *StrStrMap) DeepCopy() interface{} {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	data := make(map[string]string, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	return NewStrStrMapFrom(data, m.mu.IsSafe())
}

// IsSubOf 检查当前映射是否是 `other` 映射的子映射。
func (m *StrStrMap) IsSubOf(other *StrStrMap) bool {
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
func (m *StrStrMap) Diff(other *StrStrMap) (addedKeys, removedKeys, updatedKeys []string) {
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
