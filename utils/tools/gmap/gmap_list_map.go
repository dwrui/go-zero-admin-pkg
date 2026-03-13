package gmap

import (
	"bytes"
	"fmt"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/empty"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/glist"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gvar"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/json"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/rwmutex"
)

// ListMap 是一个保持插入顺序的映射。
//
// 它由哈希表存储值，双向链表存储顺序。
//
// 结构不是线程安全的。
//
// 参考：http://en.wikipedia.org/wiki/Associative_array
type ListMap struct {
	mu   rwmutex.RWMutex
	data map[interface{}]*glist.Element
	list *glist.List
}

type gListMapNode struct {
	key   interface{}
	value interface{}
}

// NewListMap 返回一个空的链表映射。
// ListMap 由哈希表存储值，双向链表存储顺序。
// 参数 `safe` 用于指定是否在并发安全的情况下使用映射，
// 默认为 false。
func NewListMap(safe ...bool) *ListMap {
	return &ListMap{
		mu:   rwmutex.Create(safe...),
		data: make(map[interface{}]*glist.Element),
		list: glist.New(),
	}
}

// NewListMapFrom 从给定的映射 `data` 返回一个链表映射。
// 注意，参数 `data` 映射将被设置为底层数据映射（没有深拷贝），
// 在改变外部映射时可能存在一些并发安全问题。
func NewListMapFrom(data map[interface{}]interface{}, safe ...bool) *ListMap {
	m := NewListMap(safe...)
	m.Sets(data)
	return m
}

// Iterator 是 IteratorAsc 的别名。
func (m *ListMap) Iterator(f func(key, value interface{}) bool) {
	m.IteratorAsc(f)
}

// IteratorAsc 以升序方式迭代映射只读，使用给定的回调函数 `f`。
// 如果 `f` 返回 true，则继续迭代；否则 false 停止。
func (m *ListMap) IteratorAsc(f func(key interface{}, value interface{}) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.list != nil {
		var node *gListMapNode
		m.list.IteratorAsc(func(e *glist.Element) bool {
			node = e.Value.(*gListMapNode)
			return f(node.key, node.value)
		})
	}
}

// IteratorDesc 以降序方式迭代映射只读，使用给定的回调函数 `f`。
// 如果 `f` 返回 true，则继续迭代；否则 false 停止。
func (m *ListMap) IteratorDesc(f func(key interface{}, value interface{}) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.list != nil {
		var node *gListMapNode
		m.list.IteratorDesc(func(e *glist.Element) bool {
			node = e.Value.(*gListMapNode)
			return f(node.key, node.value)
		})
	}
}

// Clone 返回一个带有当前映射数据副本的新链表映射。
func (m *ListMap) Clone(safe ...bool) *ListMap {
	return NewListMapFrom(m.Map(), safe...)
}

// Clear 删除映射的所有数据，它将重新创建一个新的底层数据映射。
func (m *ListMap) Clear() {
	m.mu.Lock()
	m.data = make(map[interface{}]*glist.Element)
	m.list = glist.New()
	m.mu.Unlock()
}

// Replace 用给定的 `data` 替换映射的数据。
func (m *ListMap) Replace(data map[interface{}]interface{}) {
	m.mu.Lock()
	m.data = make(map[interface{}]*glist.Element)
	m.list = glist.New()
	for key, value := range data {
		if e, ok := m.data[key]; !ok {
			m.data[key] = m.list.PushBack(&gListMapNode{key, value})
		} else {
			e.Value = &gListMapNode{key, value}
		}
	}
	m.mu.Unlock()
}

// Map 返回映射底层数据的副本。
func (m *ListMap) Map() map[interface{}]interface{} {
	m.mu.RLock()
	var node *gListMapNode
	var data map[interface{}]interface{}
	if m.list != nil {
		data = make(map[interface{}]interface{}, len(m.data))
		m.list.IteratorAsc(func(e *glist.Element) bool {
			node = e.Value.(*gListMapNode)
			data[node.key] = node.value
			return true
		})
	}
	m.mu.RUnlock()
	return data
}

// MapStrAny 返回映射底层数据的副本，作为 map[string]interface{}。
func (m *ListMap) MapStrAny() map[string]interface{} {
	m.mu.RLock()
	var node *gListMapNode
	var data map[string]interface{}
	if m.list != nil {
		data = make(map[string]interface{}, len(m.data))
		m.list.IteratorAsc(func(e *glist.Element) bool {
			node = e.Value.(*gListMapNode)
			data[gconv.String(node.key)] = node.value
			return true
		})
	}
	m.mu.RUnlock()
	return data
}

// FilterEmpty 删除所有值为空的键值对。
func (m *ListMap) FilterEmpty() {
	m.mu.Lock()
	if m.list != nil {
		var (
			keys = make([]interface{}, 0)
			node *gListMapNode
		)
		m.list.IteratorAsc(func(e *glist.Element) bool {
			node = e.Value.(*gListMapNode)
			if empty.IsEmpty(node.value) {
				keys = append(keys, node.key)
			}
			return true
		})
		if len(keys) > 0 {
			for _, key := range keys {
				if e, ok := m.data[key]; ok {
					delete(m.data, key)
					m.list.Remove(e)
				}
			}
		}
	}
	m.mu.Unlock()
}

// Set 设置键值到映射。
func (m *ListMap) Set(key interface{}, value interface{}) {
	m.mu.Lock()
	if m.data == nil {
		m.data = make(map[interface{}]*glist.Element)
		m.list = glist.New()
	}
	if e, ok := m.data[key]; !ok {
		m.data[key] = m.list.PushBack(&gListMapNode{key, value})
	} else {
		e.Value = &gListMapNode{key, value}
	}
	m.mu.Unlock()
}

// Sets 批量设置键值到映射。
func (m *ListMap) Sets(data map[interface{}]interface{}) {
	m.mu.Lock()
	if m.data == nil {
		m.data = make(map[interface{}]*glist.Element)
		m.list = glist.New()
	}
	for key, value := range data {
		if e, ok := m.data[key]; !ok {
			m.data[key] = m.list.PushBack(&gListMapNode{key, value})
		} else {
			e.Value = &gListMapNode{key, value}
		}
	}
	m.mu.Unlock()
}

// Search 用给定的 `key` 搜索映射。
// 第二个返回参数 `found` 如果找到键则为 true，否则为 false。
func (m *ListMap) Search(key interface{}) (value interface{}, found bool) {
	m.mu.RLock()
	if m.data != nil {
		if e, ok := m.data[key]; ok {
			value = e.Value.(*gListMapNode).value
			found = ok
		}
	}
	m.mu.RUnlock()
	return
}

// Get 通过给定的 `key` 返回值。
func (m *ListMap) Get(key interface{}) (value interface{}) {
	m.mu.RLock()
	if m.data != nil {
		if e, ok := m.data[key]; ok {
			value = e.Value.(*gListMapNode).value
		}
	}
	m.mu.RUnlock()
	return
}

// Pop 从映射中检索并删除一个项目。
func (m *ListMap) Pop() (key, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, e := range m.data {
		value = e.Value.(*gListMapNode).value
		delete(m.data, k)
		m.list.Remove(e)
		return k, value
	}
	return
}

// Pops 从映射中检索并删除 `size` 个项目。
// 如果 size == -1，则返回所有项目。
func (m *ListMap) Pops(size int) map[interface{}]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if size > len(m.data) || size == -1 {
		size = len(m.data)
	}
	if size == 0 {
		return nil
	}
	index := 0
	newMap := make(map[interface{}]interface{}, size)
	for k, e := range m.data {
		value := e.Value.(*gListMapNode).value
		delete(m.data, k)
		m.list.Remove(e)
		newMap[k] = value
		index++
		if index == size {
			break
		}
	}
	return newMap
}

// doSetWithLockCheck 用 mutex.Lock 检查键的值是否存在，
// 如果不存在，用给定的 `key` 设置值到映射，
// 否则只返回现有值。
//
// 设置值时，如果 `value` 是 `func() interface {}` 类型，
// 它将在映射的 mutex.Lock 下执行，
// 其返回值将用 `key` 设置到映射。
//
// 它返回给定 `key` 的值。
func (m *ListMap) doSetWithLockCheck(key interface{}, value interface{}) interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[interface{}]*glist.Element)
		m.list = glist.New()
	}
	if e, ok := m.data[key]; ok {
		return e.Value.(*gListMapNode).value
	}
	if f, ok := value.(func() interface{}); ok {
		value = f()
	}
	if value != nil {
		m.data[key] = m.list.PushBack(&gListMapNode{key, value})
	}
	return value
}

// GetOrSet 通过键返回值，
// 或者如果它不存在则用给定的 `value` 设置值，然后返回这个值。
func (m *ListMap) GetOrSet(key interface{}, value interface{}) interface{} {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, value)
	} else {
		return v
	}
}

// GetOrSetFunc 通过键返回值，
// 或者如果它不存在则用回调函数 `f` 的返回值设置值
// 然后返回这个值。
func (m *ListMap) GetOrSetFunc(key interface{}, f func() interface{}) interface{} {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, f())
	} else {
		return v
	}
}

// GetOrSetFuncLock 通过键返回值，
// 或者如果它不存在则用回调函数 `f` 的返回值设置值
// 然后返回这个值。
//
// GetOrSetFuncLock 与 GetOrSetFunc 函数的不同之处在于它执行函数 `f`
// 在映射的 mutex.Lock 下。
func (m *ListMap) GetOrSetFuncLock(key interface{}, f func() interface{}) interface{} {
	if v, ok := m.Search(key); !ok {
		return m.doSetWithLockCheck(key, f)
	} else {
		return v
	}
}

// GetVar 返回一个带有给定 `key` 值的 Var。
// 返回的 Var 是非并发安全的。
func (m *ListMap) GetVar(key interface{}) *gvar.Var {
	return gvar.New(m.Get(key))
}

// GetVarOrSet 返回一个带有 GetVarOrSet 结果的 Var。
// 返回的 Var 是非并发安全的。
func (m *ListMap) GetVarOrSet(key interface{}, value interface{}) *gvar.Var {
	return gvar.New(m.GetOrSet(key, value))
}

// GetVarOrSetFunc 返回一个带有 GetOrSetFunc 结果的 Var。
// 返回的 Var 是非并发安全的。
func (m *ListMap) GetVarOrSetFunc(key interface{}, f func() interface{}) *gvar.Var {
	return gvar.New(m.GetOrSetFunc(key, f))
}

// GetVarOrSetFuncLock 返回一个带有 GetOrSetFuncLock 结果的 Var。
// 返回的 Var 是非并发安全的。
func (m *ListMap) GetVarOrSetFuncLock(key interface{}, f func() interface{}) *gvar.Var {
	return gvar.New(m.GetOrSetFuncLock(key, f))
}

// SetIfNotExist 如果 `key` 不存在则设置 `value` 到映射，然后返回 true。
// 如果 `key` 存在则返回 false，并且 `value` 将被忽略。
func (m *ListMap) SetIfNotExist(key interface{}, value interface{}) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, value)
		return true
	}
	return false
}

// SetIfNotExistFunc 用回调函数 `f` 的返回值设置值，然后返回 true。
// 如果 `key` 存在则返回 false，并且 `value` 将被忽略。
func (m *ListMap) SetIfNotExistFunc(key interface{}, f func() interface{}) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, f())
		return true
	}
	return false
}

// SetIfNotExistFuncLock 用回调函数 `f` 的返回值设置值，然后返回 true。
// 如果 `key` 存在则返回 false，并且 `value` 将被忽略。
//
// SetIfNotExistFuncLock 与 SetIfNotExistFunc 函数的不同之处在于
// 它在映射的 mutex.Lock 下执行函数 `f`。
func (m *ListMap) SetIfNotExistFuncLock(key interface{}, f func() interface{}) bool {
	if !m.Contains(key) {
		m.doSetWithLockCheck(key, f)
		return true
	}
	return false
}

// Remove 通过给定的 `key` 从映射中删除值，并返回这个删除的值。
func (m *ListMap) Remove(key interface{}) (value interface{}) {
	m.mu.Lock()
	if m.data != nil {
		if e, ok := m.data[key]; ok {
			value = e.Value.(*gListMapNode).value
			delete(m.data, key)
			m.list.Remove(e)
		}
	}
	m.mu.Unlock()
	return
}

// Removes 通过键批量删除映射的值。
func (m *ListMap) Removes(keys []interface{}) {
	m.mu.Lock()
	if m.data != nil {
		for _, key := range keys {
			if e, ok := m.data[key]; ok {
				delete(m.data, key)
				m.list.Remove(e)
			}
		}
	}
	m.mu.Unlock()
}

// Keys 返回映射的所有键作为升序切片。
func (m *ListMap) Keys() []interface{} {
	m.mu.RLock()
	var (
		keys  = make([]interface{}, m.list.Len())
		index = 0
	)
	if m.list != nil {
		m.list.IteratorAsc(func(e *glist.Element) bool {
			keys[index] = e.Value.(*gListMapNode).key
			index++
			return true
		})
	}
	m.mu.RUnlock()
	return keys
}

// Values 返回映射的所有值作为切片。
func (m *ListMap) Values() []interface{} {
	m.mu.RLock()
	var (
		values = make([]interface{}, m.list.Len())
		index  = 0
	)
	if m.list != nil {
		m.list.IteratorAsc(func(e *glist.Element) bool {
			values[index] = e.Value.(*gListMapNode).value
			index++
			return true
		})
	}
	m.mu.RUnlock()
	return values
}

// Contains 检查键是否存在。
// 如果 `key` 存在则返回 true，否则返回 false。
func (m *ListMap) Contains(key interface{}) (ok bool) {
	m.mu.RLock()
	if m.data != nil {
		_, ok = m.data[key]
	}
	m.mu.RUnlock()
	return
}

// Size 返回映射的大小。
func (m *ListMap) Size() (size int) {
	m.mu.RLock()
	size = len(m.data)
	m.mu.RUnlock()
	return
}

// IsEmpty 检查映射是否为空。
// 如果映射为空则返回 true，否则返回 false。
func (m *ListMap) IsEmpty() bool {
	return m.Size() == 0
}

// Flip 交换映射的键值到值键。
func (m *ListMap) Flip() {
	data := m.Map()
	m.Clear()
	for key, value := range data {
		m.Set(value, key)
	}
}

// Merge 合并两个链表映射。
// `other` 映射将被合并到映射 `m` 中。
func (m *ListMap) Merge(other *ListMap) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[interface{}]*glist.Element)
		m.list = glist.New()
	}
	if other != m {
		other.mu.RLock()
		defer other.mu.RUnlock()
	}
	var node *gListMapNode
	other.list.IteratorAsc(func(e *glist.Element) bool {
		node = e.Value.(*gListMapNode)
		if e, ok := m.data[node.key]; !ok {
			m.data[node.key] = m.list.PushBack(&gListMapNode{node.key, node.value})
		} else {
			e.Value = &gListMapNode{node.key, node.value}
		}
		return true
	})
}

// String 返回映射作为字符串。
func (m *ListMap) String() string {
	if m == nil {
		return ""
	}
	b, _ := m.MarshalJSON()
	return string(b)
}

// MarshalJSON 实现 json.Marshal 的 MarshalJSON 接口。
func (m ListMap) MarshalJSON() (jsonBytes []byte, err error) {
	if m.data == nil {
		return []byte("null"), nil
	}
	buffer := bytes.NewBuffer(nil)
	buffer.WriteByte('{')
	m.Iterator(func(key, value interface{}) bool {
		valueBytes, valueJsonErr := json.Marshal(value)
		if valueJsonErr != nil {
			err = valueJsonErr
			return false
		}
		if buffer.Len() > 1 {
			buffer.WriteByte(',')
		}
		buffer.WriteString(fmt.Sprintf(`"%v":%s`, key, valueBytes))
		return true
	})
	buffer.WriteByte('}')
	return buffer.Bytes(), nil
}

// UnmarshalJSON 实现 json.Unmarshal 的 UnmarshalJSON 接口。
func (m *ListMap) UnmarshalJSON(b []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[interface{}]*glist.Element)
		m.list = glist.New()
	}
	var data map[string]interface{}
	if err := json.UnmarshalUseNumber(b, &data); err != nil {
		return err
	}
	for key, value := range data {
		if e, ok := m.data[key]; !ok {
			m.data[key] = m.list.PushBack(&gListMapNode{key, value})
		} else {
			e.Value = &gListMapNode{key, value}
		}
	}
	return nil
}

// UnmarshalValue 是一个接口实现，为映射设置任何类型的值。
func (m *ListMap) UnmarshalValue(value interface{}) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[interface{}]*glist.Element)
		m.list = glist.New()
	}
	for k, v := range gconv.Map(value) {
		if e, ok := m.data[k]; !ok {
			m.data[k] = m.list.PushBack(&gListMapNode{k, v})
		} else {
			e.Value = &gListMapNode{k, v}
		}
	}
	return
}
