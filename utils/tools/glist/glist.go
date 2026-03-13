// 包glist提供了最常用的双向链表容器，该容器还支持
// 并发安全/不安全切换特性。
package glist

import (
	"bytes"
	"container/list"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/json"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/rwmutex"
)

type (
	// List 是一个包含并发安全/不安全开关的双向链表。
	// 该开关应在初始化时设置，之后不能更改。
	List struct {
		mu   rwmutex.RWMutex
		list *list.List
	}
	// Element 是列表的元素类型。
	Element = list.Element
)

// New 创建并返回一个新的空双向链表。
func New(safe ...bool) *List {
	return &List{
		mu:   rwmutex.Create(safe...),
		list: list.New(),
	}
}

// NewFrom 从给定切片 `array` 的副本创建并返回一个列表。
// 参数 `safe` 用于指定是否使用并发安全，默认为 false。
func NewFrom(array []interface{}, safe ...bool) *List {
	l := list.New()
	for _, v := range array {
		l.PushBack(v)
	}
	return &List{
		mu:   rwmutex.Create(safe...),
		list: l,
	}
}

// PushFront 在列表 `l` 的前端插入一个新元素 `e`，其值为 `v`，并返回 `e`。
func (l *List) PushFront(v interface{}) (e *Element) {
	l.mu.Lock()
	if l.list == nil {
		l.list = list.New()
	}
	e = l.list.PushFront(v)
	l.mu.Unlock()
	return
}

// PushBack 在列表 `l` 的后端插入一个新元素 `e`，其值为 `v`，并返回 `e`。
func (l *List) PushBack(v interface{}) (e *Element) {
	l.mu.Lock()
	if l.list == nil {
		l.list = list.New()
	}
	e = l.list.PushBack(v)
	l.mu.Unlock()
	return
}

// PushFronts 在列表 `l` 的前端插入多个新元素，其值为 `values`。
func (l *List) PushFronts(values []interface{}) {
	l.mu.Lock()
	if l.list == nil {
		l.list = list.New()
	}
	for _, v := range values {
		l.list.PushFront(v)
	}
	l.mu.Unlock()
}

// PushBacks 在列表 `l` 的后端插入多个新元素，其值为 `values`。
func (l *List) PushBacks(values []interface{}) {
	l.mu.Lock()
	if l.list == nil {
		l.list = list.New()
	}
	for _, v := range values {
		l.list.PushBack(v)
	}
	l.mu.Unlock()
}

// PopBack 从 `l` 的后端移除元素并返回该元素的值。
func (l *List) PopBack() (value interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
		return
	}
	if e := l.list.Back(); e != nil {
		value = l.list.Remove(e)
	}
	return
}

// PopFront 从 `l` 的前端移除元素并返回该元素的值。
func (l *List) PopFront() (value interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
		return
	}
	if e := l.list.Front(); e != nil {
		value = l.list.Remove(e)
	}
	return
}

// PopBacks 从 `l` 的后端移除 `max` 个元素，
// 并返回被移除元素的值作为切片。
func (l *List) PopBacks(max int) (values []interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
		return
	}
	length := l.list.Len()
	if length > 0 {
		if max > 0 && max < length {
			length = max
		}
		values = make([]interface{}, length)
		for i := 0; i < length; i++ {
			values[i] = l.list.Remove(l.list.Back())
		}
	}
	return
}

// PopFronts 从 `l` 的前端移除 `max` 个元素，
// 并返回被移除元素的值作为切片。
func (l *List) PopFronts(max int) (values []interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
		return
	}
	length := l.list.Len()
	if length > 0 {
		if max > 0 && max < length {
			length = max
		}
		values = make([]interface{}, length)
		for i := 0; i < length; i++ {
			values[i] = l.list.Remove(l.list.Front())
		}
	}
	return
}

// PopBackAll 从 `l` 的后端移除所有元素，
// 并返回被移除元素的值作为切片。
func (l *List) PopBackAll() []interface{} {
	return l.PopBacks(-1)
}

// PopFrontAll 从 `l` 的前端移除所有元素，
// 并返回被移除元素的值作为切片。
func (l *List) PopFrontAll() []interface{} {
	return l.PopFronts(-1)
}

// FrontAll 复制并返回 `l` 前端所有元素的值作为切片。
func (l *List) FrontAll() (values []interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.list == nil {
		return
	}
	length := l.list.Len()
	if length > 0 {
		values = make([]interface{}, length)
		for i, e := 0, l.list.Front(); i < length; i, e = i+1, e.Next() {
			values[i] = e.Value
		}
	}
	return
}

// BackAll 复制并返回 `l` 后端所有元素的值作为切片。
func (l *List) BackAll() (values []interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.list == nil {
		return
	}
	length := l.list.Len()
	if length > 0 {
		values = make([]interface{}, length)
		for i, e := 0, l.list.Back(); i < length; i, e = i+1, e.Prev() {
			values[i] = e.Value
		}
	}
	return
}

// FrontValue 返回 `l` 第一个元素的值，如果列表为空则返回 nil。
func (l *List) FrontValue() (value interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.list == nil {
		return
	}
	if e := l.list.Front(); e != nil {
		value = e.Value
	}
	return
}

// BackValue 返回 `l` 最后一个元素的值，如果列表为空则返回 nil。
func (l *List) BackValue() (value interface{}) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.list == nil {
		return
	}
	if e := l.list.Back(); e != nil {
		value = e.Value
	}
	return
}

// Front 返回列表 `l` 的第一个元素，如果列表为空则返回 nil。
func (l *List) Front() (e *Element) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.list == nil {
		return
	}
	e = l.list.Front()
	return
}

// Back 返回列表 `l` 的最后一个元素，如果列表为空则返回 nil。
func (l *List) Back() (e *Element) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.list == nil {
		return
	}
	e = l.list.Back()
	return
}

// Len 返回列表 `l` 的元素数量。
// 复杂度为 O(1)。
func (l *List) Len() (length int) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.list == nil {
		return
	}
	length = l.list.Len()
	return
}

// Size 是 Len 的别名。
func (l *List) Size() int {
	return l.Len()
}

// MoveBefore 将元素 `e` 移动到 `p` 之前的新位置。
// 如果 `e` 或 `p` 不是 `l` 的元素，或者 `e` == `p`，则列表不被修改。
// 元素和 `p` 不能为 nil。
func (l *List) MoveBefore(e, p *Element) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	l.list.MoveBefore(e, p)
}

// MoveAfter 将元素 `e` 移动到 `p` 之后的新位置。
// 如果 `e` 或 `p` 不是 `l` 的元素，或者 `e` == `p`，则列表不被修改。
// 元素和 `p` 不能为 nil。
func (l *List) MoveAfter(e, p *Element) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	l.list.MoveAfter(e, p)
}

// MoveToFront 将元素 `e` 移动到列表 `l` 的前端。
// 如果 `e` 不是 `l` 的元素，则列表不被修改。
// 元素不能为 nil。
func (l *List) MoveToFront(e *Element) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	l.list.MoveToFront(e)
}

// MoveToBack 将元素 `e` 移动到列表 `l` 的后端。
// 如果 `e` 不是 `l` 的元素，则列表不被修改。
// 元素不能为 nil。
func (l *List) MoveToBack(e *Element) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	l.list.MoveToBack(e)
}

// PushBackList 在列表 `l` 的后端插入另一个列表的副本。
// 列表 `l` 和 `other` 可以相同，但不能为 nil。
func (l *List) PushBackList(other *List) {
	if l != other {
		other.mu.RLock()
		defer other.mu.RUnlock()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	l.list.PushBackList(other.list)
}

// PushFrontList 在列表 `l` 的前端插入另一个列表的副本。
// 列表 `l` 和 `other` 可以相同，但不能为 nil。
func (l *List) PushFrontList(other *List) {
	if l != other {
		other.mu.RLock()
		defer other.mu.RUnlock()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	l.list.PushFrontList(other.list)
}

// InsertAfter 在 `p` 之后立即插入一个新元素 `e`，其值为 `v`，并返回 `e`。
// 如果 `p` 不是 `l` 的元素，则列表不被修改。
// `p` 不能为 nil。
func (l *List) InsertAfter(p *Element, v interface{}) (e *Element) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	e = l.list.InsertAfter(v, p)
	return
}

// InsertBefore 在 `p` 之前立即插入一个新元素 `e`，其值为 `v`，并返回 `e`。
// 如果 `p` 不是 `l` 的元素，则列表不被修改。
// `p` 不能为 nil。
func (l *List) InsertBefore(p *Element, v interface{}) (e *Element) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	e = l.list.InsertBefore(v, p)
	return
}

// Remove 如果 `e` 是列表 `l` 的元素，则从 `l` 中移除 `e`。
// 它返回元素值 e.Value。
// 元素不能为 nil。
func (l *List) Remove(e *Element) (value interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	value = l.list.Remove(e)
	return
}

// Removes 从 `l` 中移除多个元素 `es`（如果 `es` 是列表 `l` 的元素）。
func (l *List) Removes(es []*Element) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	for _, e := range es {
		l.list.Remove(e)
	}
}

// RemoveAll 从列表 `l` 中移除所有元素。
func (l *List) RemoveAll() {
	l.mu.Lock()
	l.list = list.New()
	l.mu.Unlock()
}

// Clear 是 RemoveAll 的别名。
func (l *List) Clear() {
	l.RemoveAll()
}

// RLockFunc 使用 RWMutex.RLock 内的给定回调函数 `f` 锁定读取。
func (l *List) RLockFunc(f func(list *list.List)) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.list != nil {
		f(l.list)
	}
}

// LockFunc 使用 RWMutex.Lock 内的给定回调函数 `f` 锁定写入。
func (l *List) LockFunc(f func(list *list.List)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	f(l.list)
}

// Iterator 是 IteratorAsc 的别名。
func (l *List) Iterator(f func(e *Element) bool) {
	l.IteratorAsc(f)
}

// IteratorAsc 使用给定的回调函数 `f` 以升序方式只读迭代列表。
// 如果 `f` 返回 true，则继续迭代；如果返回 false，则停止。
func (l *List) IteratorAsc(f func(e *Element) bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.list == nil {
		return
	}
	length := l.list.Len()
	if length > 0 {
		for i, e := 0, l.list.Front(); i < length; i, e = i+1, e.Next() {
			if !f(e) {
				break
			}
		}
	}
}

// IteratorDesc 使用给定的回调函数 `f` 以降序方式只读迭代列表。
// 如果 `f` 返回 true，则继续迭代；如果返回 false，则停止。
func (l *List) IteratorDesc(f func(e *Element) bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.list == nil {
		return
	}
	length := l.list.Len()
	if length > 0 {
		for i, e := 0, l.list.Back(); i < length; i, e = i+1, e.Prev() {
			if !f(e) {
				break
			}
		}
	}
}

// Join 使用字符串 `glue` 连接列表元素。
func (l *List) Join(glue string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.list == nil {
		return ""
	}
	buffer := bytes.NewBuffer(nil)
	length := l.list.Len()
	if length > 0 {
		for i, e := 0, l.list.Front(); i < length; i, e = i+1, e.Next() {
			buffer.WriteString(gconv.String(e.Value))
			if i != length-1 {
				buffer.WriteString(glue)
			}
		}
	}
	return buffer.String()
}

// String 返回当前列表作为字符串。
func (l *List) String() string {
	if l == nil {
		return ""
	}
	return "[" + l.Join(",") + "]"
}

// MarshalJSON 实现 json.Marshal 的 MarshalJSON 接口。
func (l List) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.FrontAll())
}

// UnmarshalJSON 实现 json.Unmarshal 的 UnmarshalJSON 接口。
func (l *List) UnmarshalJSON(b []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	var array []interface{}
	if err := json.UnmarshalUseNumber(b, &array); err != nil {
		return err
	}
	l.PushBacks(array)
	return nil
}

// UnmarshalValue 是一个接口实现，它为列表设置任何类型的值。
func (l *List) UnmarshalValue(value interface{}) (err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.list == nil {
		l.list = list.New()
	}
	var array []interface{}
	switch value.(type) {
	case string, []byte:
		err = json.UnmarshalUseNumber(gconv.Bytes(value), &array)
	default:
		array = gconv.SliceAny(value)
	}
	l.PushBacks(array)
	return err
}
