package gmap

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/empty"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gvar"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/json"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/rwmutex"
	"reflect"
)

// IntAnyMap 实现了带有 switch 的 RWMutex 的 map[int]interface{}。
type IntAnyMap struct {
	mu   rwmutex.RWMutex
	data map[int]interface{}
}

// NewIntAnyMap 返回一个空的 IntAnyMap 对象。
// 参数“安全”用于指定是否在并发安全中使用映射，
// 默认情况下是false。
func NewIntAnyMap(safe ...bool) *IntAnyMap {
	return &IntAnyMap{
		mu:   rwmutex.Create(safe...),
		data: make(map[int]interface{}),
	}
}

// NewIntAnyMapFrom 创建并返回一个从给定map `data`创建的哈希映射。
// 请注意，参数 `data` 映射将被设置为底层数据映射（无深拷贝），
// 因此在外部更改映射时可能会出现并发安全问题。
// 参数“安全”用于指定是否在并发安全中使用映射，
// 默认情况下是false。
func NewIntAnyMapFrom(data map[int]interface{}, safe ...bool) *IntAnyMap {
	return &IntAnyMap{
		mu:   rwmutex.Create(safe...),
		data: data,
	}
}

// Iterator 遍历哈希映射 readonly 并使用自定义回调函数 `f`。
// 如果 `f` 返回 true，则继续迭代；否则停止迭代。
func (m *IntAnyMap) Iterator(f func(k int, v interface{}) bool) {
	for k, v := range m.Map() {
		if !f(k, v) {
			break
		}
	}
}

// Clone 返回一个新的哈希映射，其中包含当前映射数据的副本。
func (m *IntAnyMap) Clone() *IntAnyMap {
	return NewIntAnyMapFrom(m.MapCopy(), m.mu.IsSafe())
}

// Map 返回哈希映射的底层数据映射。
// 请注意，如果在并发安全使用中，它返回底层数据的副本，
// 否则返回指向底层数据的指针。
func (m *IntAnyMap) Map() map[int]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.mu.IsSafe() {
		return m.data
	}
	data := make(map[int]interface{}, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	return data
}

// MapStrAny 返回地图底层数据的副本，作为 map[string]interface{}。
func (m *IntAnyMap) MapStrAny() map[string]interface{} {
	m.mu.RLock()
	data := make(map[string]interface{}, len(m.data))
	for k, v := range m.data {
		data[gconv.String(k)] = v
	}
	m.mu.RUnlock()
	return data
}

// MapCopy 返回哈希映射底层数据的副本。
func (m *IntAnyMap) MapCopy() map[int]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data := make(map[int]interface{}, len(m.data))
	for k, v := range m.data {
		data[k] = v
	}
	return data
}

// FilterEmpty 删除所有键值为空的键值对。
// 诸如：0、nil、false、“”、len（slice/map/chan） == 0 等值被视为空。
func (m *IntAnyMap) FilterEmpty() {
	m.mu.Lock()
	for k, v := range m.data {
		if empty.IsEmpty(v) {
			delete(m.data, k)
		}
	}
	m.mu.Unlock()
}

// FilterNil 删除所有键值为 nil 的键值对。
func (m *IntAnyMap) FilterNil() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range m.data {
		if empty.IsNil(v) {
			delete(m.data, k)
		}
	}
}

// Set 将键值设置为哈希映射。
func (m *IntAnyMap) Set(key int, val interface{}) {
	m.mu.Lock()
	if m.data == nil {
		m.data = make(map[int]interface{})
	}
	m.data[key] = val
	m.mu.Unlock()
}

// 批量设置键值到哈希映射。
func (m *IntAnyMap) Sets(data map[int]interface{}) {
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

// 搜索用给定的“键”搜索地图。
// 第二个返回参数“found”如果找到key则为真，否则为false。
func (m *IntAnyMap) Search(key int) (value interface{}, found bool) {
	m.mu.RLock()
	if m.data != nil {
		value, found = m.data[key]
	}
	m.mu.RUnlock()
	return
}

// get 通过给定的“key”返回该值。
func (m *IntAnyMap) Get(key int) (value interface{}) {
	m.mu.RLock()
	if m.data != nil {
		value = m.data[key]
	}
	m.mu.RUnlock()
	return
}

// Pop 从map上取回和删除一个项目。
func (m *IntAnyMap) Pop() (key int, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, value = range m.data {
		delete(m.data, key)
		return
	}
	return
}

// Pops 从map上取回并删除`size`个项目。
// 如果size == -1，则返回所有项目。
func (m *IntAnyMap) Pops(size int) map[int]interface{} {
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
		newMap = make(map[int]interface{}, size)
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

// doSetWithLockCheck 检查是否存在“key”的值，
// 如果不存在，则将“value”设置为“key”，
// 否则仅返回现有值。
//
// 当设置值时，如果“value”是“func() interface {}”类型，
// 它将与哈希映射的mutex.Lock执行，
// 并将其返回值设置为“key”。
//
// 它返回给定“key”的值。
func (m *IntAnyMap) doSetWithLockCheck(key int, value interface{}) interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[int]interface{})
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

// GetOrSet 返回给定“key”的值，
// 如果不存在，则将“value”设置为“key”，
// 然后返回此值。
func (m *IntAnyMap) GetOrSet(key int, value interface{}) interface{} {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, value)
	} else {
		return v
	}
}

// GetOrSetFunc 返回给定“key”的值，
// 如果不存在，则将“value”设置为“key”，
// 然后返回此值。
func (m *IntAnyMap) GetOrSetFunc(key int, f func() interface{}) interface{} {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, f())
	} else {
		return v
	}
}

// GetOrSetFuncLock 返回给定“key”的值，
// 如果不存在，则将“value”设置为“key”，
// 然后返回此值。
//
// GetOrSetFuncLock 与 GetOrSetFunc 函数的不同之处在于，
// 它使用哈希映射的 mutex.Lock 执行函数“f”。
func (m *IntAnyMap) GetOrSetFuncLock(key int, f func() interface{}) interface{} {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, f)
	} else {
		return v
	}
}

// GetVar 返回给定“key”的值的 Var 对象。
// 返回的 Var 对象不是并发安全的。
func (m *IntAnyMap) GetVar(key int) *gvar.Var {
	return gvar.New(m.Get(key))
}

// GetVarOrSet 返回给定“key”的值的 Var 对象。
// 返回的 Var 对象不是并发安全的。
func (m *IntAnyMap) GetVarOrSet(key int, value interface{}) *gvar.Var {
	return gvar.New(m.GetOrSet(key, value))
}

// GetVarOrSetFunc 返回给定“key”的值的 Var 对象。
// 返回的 Var 对象不是并发安全的。
func (m *IntAnyMap) GetVarOrSetFunc(key int, f func() interface{}) *gvar.Var {
	return gvar.New(m.GetOrSetFunc(key, f))
}

// GetVarOrSetFuncLock 返回给定“key”的值的 Var 对象。
// 返回的 Var 对象不是并发安全的。
func (m *IntAnyMap) GetVarOrSetFuncLock(key int, f func() interface{}) *gvar.Var {
	return gvar.New(m.GetOrSetFuncLock(key, f))
}

// SetIfNotExist 设置“value”到哈希映射中，如果“key”不存在，则返回 true。
// 如果“key”存在，则返回 false，并且“value”将被忽略。
func (m *IntAnyMap) SetIfNotExist(key int, value interface{}) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, value)
		return true
	}
	return false
}

// SetIfNotExistFunc 设置“value”到哈希映射中，如果“key”不存在，则返回 true。
// 如果“key”存在，则返回 false，并且“value”将被忽略。
func (m *IntAnyMap) SetIfNotExistFunc(key int, f func() interface{}) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, f())
		return true
	}
	return false
}

// SetIfNotExistFuncLock 设置“value”到哈希映射中，如果“key”不存在，则返回 true。
// 如果“key”存在，则返回 false，并且“value”将被忽略。
//
// SetIfNotExistFuncLock 与 SetIfNotExistFunc 函数的不同之处在于，
// 它使用哈希映射的 mutex.Lock 执行函数“f”。
func (m *IntAnyMap) SetIfNotExistFuncLock(key int, f func() interface{}) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, f)
		return true
	}
	return false
}

// Removes 批量删除哈希映射中给定“keys”的值。
func (m *IntAnyMap) Removes(keys []int) {
	m.mu.Lock()
	if m.data != nil {
		for _, key := range keys {
			delete(m.data, key)
		}
	}
	m.mu.Unlock()
}

// Remove 删除哈希映射中给定“key”的值，并返回此删除的值。
func (m *IntAnyMap) Remove(key int) (value interface{}) {
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

// Keys 返回哈希映射中所有键的切片。
func (m *IntAnyMap) Keys() []int {
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

// Values 返回哈希映射中所有值的切片。
func (m *IntAnyMap) Values() []interface{} {
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

// Contains 检查哈希映射中是否存在“key”。
// 如果“key”存在，则返回 true，否则返回 false。
func (m *IntAnyMap) Contains(key int) bool {
	var ok bool
	m.mu.RLock()
	if m.data != nil {
		_, ok = m.data[key]
	}
	m.mu.RUnlock()
	return ok
}

// Size 返回哈希映射中元素的数量。
func (m *IntAnyMap) Size() int {
	m.mu.RLock()
	length := len(m.data)
	m.mu.RUnlock()
	return length
}

// IsEmpty 检查哈希映射是否为空。
// 如果哈希映射为空，则返回 true，否则返回 false。
func (m *IntAnyMap) IsEmpty() bool {
	return m.Size() == 0
}

// Clear 删除哈希映射中的所有数据，它将重新创建一个新的底层数据映射。
func (m *IntAnyMap) Clear() {
	m.mu.Lock()
	m.data = make(map[int]interface{})
	m.mu.Unlock()
}

// Replace 用给定的 `data` 替换哈希映射中的所有数据。
func (m *IntAnyMap) Replace(data map[int]interface{}) {
	m.mu.Lock()
	m.data = data
	m.mu.Unlock()
}

// LockFunc 用给定的回调函数 `f` 锁定写入操作，在 RWMutex.Lock 中执行。
func (m *IntAnyMap) LockFunc(f func(m map[int]interface{})) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f(m.data)
}

// RLockFunc 用给定的回调函数 `f` 锁定读取操作，在 RWMutex.RLock 中执行。
func (m *IntAnyMap) RLockFunc(f func(m map[int]interface{})) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f(m.data)
}

// Flip 交换哈希映射中的键值对，将值作为键，将键作为值。
func (m *IntAnyMap) Flip() {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := make(map[int]interface{}, len(m.data))
	for k, v := range m.data {
		n[gconv.Int(v)] = k
	}
	m.data = n
}

// Merge 合并两个哈希映射。
// 参数 `other` 中的键值对将被合并到哈希映射 `m` 中。
func (m *IntAnyMap) Merge(other *IntAnyMap) {
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

// String 返回哈希映射的字符串表示形式。
func (m *IntAnyMap) String() string {
	if m == nil {
		return ""
	}
	b, _ := m.MarshalJSON()
	return string(b)
}

// MarshalJSON 实现了 json.Marshal 接口，将哈希映射序列化为 JSON 字符串。
func (m IntAnyMap) MarshalJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return json.Marshal(m.data)
}

// UnmarshalJSON 实现了 json.Unmarshal 接口，将 JSON 字符串反序列化为哈希映射。
func (m *IntAnyMap) UnmarshalJSON(b []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[int]interface{})
	}
	if err := json.UnmarshalUseNumber(b, &m.data); err != nil {
		return err
	}
	return nil
}

// UnmarshalValue 实现了 json.Unmarshal 接口，将 JSON 字符串反序列化为哈希映射。
func (m *IntAnyMap) UnmarshalValue(value interface{}) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[int]interface{})
	}
	switch value.(type) {
	case string, []byte:
		return json.UnmarshalUseNumber(gconv.Bytes(value), &m.data)
	default:
		for k, v := range gconv.Map(value) {
			m.data[gconv.Int(k)] = v
		}
	}
	return
}

// IsSubOf 检查当前哈希映射是否是 `other` 哈希映射的子映射。
// 如果当前哈希映射中的所有键值对都存在于 `other` 哈希映射中，且对应的值相等，则返回 true，否则返回 false。
func (m *IntAnyMap) IsSubOf(other *IntAnyMap) bool {
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

// Diff 比较当前哈希映射 `m` 与哈希映射 `other` 并返回它们不同的键。
// 返回的 `addedKeys` 是在哈希映射 `m` 中但不在哈希映射 `other` 中的键。
// 返回的 `removedKeys` 是在哈希映射 `other` 中但不在哈希映射 `m` 中的键。
// 返回的 `updatedKeys` 是在哈希映射 `m` 和 `other` 中都存在的键，但对应的值不相等（`!=`）。
func (m *IntAnyMap) Diff(other *IntAnyMap) (addedKeys, removedKeys, updatedKeys []int) {
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
