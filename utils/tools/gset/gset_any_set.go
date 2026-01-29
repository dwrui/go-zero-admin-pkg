// 包 gset 提供了各种并发安全/非安全集合。
package gset

import (
	"bytes"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gstr"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/json"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/rwmutex"
)

// Set 是一个由 interface{} 项组成的集合。
type Set struct {
	mu   rwmutex.RWMutex
	data map[interface{}]struct{}
}

// New 创建并返回一个新的集合，该集合包含不重复的项。
// 参数 `safe` 用于指定是否在并发安全模式下使用集合，
// 默认值为 false。
func New(safe ...bool) *Set {
	return NewSet(safe...)
}

// NewSet 创建并返回一个新的集合，该集合包含不重复的项。
// 参数 `safe` 用于指定是否在并发安全模式下使用集合，
// 默认值为 false。
func NewSet(safe ...bool) *Set {
	return &Set{
		data: make(map[interface{}]struct{}),
		mu:   rwmutex.Create(safe...),
	}
}

// NewFrom 返回一个新的集合，该集合包含 `items` 中的不重复项。
// 参数 `items` 可以是任意类型的变量，也可以是切片。
func NewFrom(items interface{}, safe ...bool) *Set {
	m := make(map[interface{}]struct{})
	for _, v := range gconv.Interfaces(items) {
		m[v] = struct{}{}
	}
	return &Set{
		data: m,
		mu:   rwmutex.Create(safe...),
	}
}

// Iterator 遍历集合中的所有项，只读模式，
// 如果回调函数 `f` 返回 true，则继续迭代；
// 否则停止迭代。
func (set *Set) Iterator(f func(v interface{}) bool) {
	for _, k := range set.Slice() {
		if !f(k) {
			break
		}
	}
}

// Add 添加一个或多个项到集合中。
func (set *Set) Add(items ...interface{}) {
	set.mu.Lock()
	if set.data == nil {
		set.data = make(map[interface{}]struct{})
	}
	for _, v := range items {
		set.data[v] = struct{}{}
	}
	set.mu.Unlock()
}

// AddIfNotExist 检查项是否存在于集合中，
// 如果项不存在于集合中，则将项添加到集合中并返回 true，
// 否则不执行任何操作并返回 false。
//
// 注意：如果 `item` 为 nil，则不执行任何操作并返回 false。
func (set *Set) AddIfNotExist(item interface{}) bool {
	if item == nil {
		return false
	}
	if !set.Contains(item) {
		set.mu.Lock()
		defer set.mu.Unlock()
		if set.data == nil {
			set.data = make(map[interface{}]struct{})
		}
		if _, ok := set.data[item]; !ok {
			set.data[item] = struct{}{}
			return true
		}
	}
	return false
}

// AddIfNotExistFunc 检查项是否存在于集合中，
// 如果项不存在于集合中且函数 `f` 返回 true，则将项添加到集合中并返回 true，
// 否则不执行任何操作并返回 false。
//
// 注意：如果 `item` 为 nil，则不执行任何操作并返回 false。
// 函数 `f` 是在读取锁之外执行的。
func (set *Set) AddIfNotExistFunc(item interface{}, f func() bool) bool {
	if item == nil {
		return false
	}
	if !set.Contains(item) {
		if f() {
			set.mu.Lock()
			defer set.mu.Unlock()
			if set.data == nil {
				set.data = make(map[interface{}]struct{})
			}
			if _, ok := set.data[item]; !ok {
				set.data[item] = struct{}{}
				return true
			}
		}
	}
	return false
}

// AddIfNotExistFuncLock 检查项是否存在于集合中，
// 如果项不存在于集合中且函数 `f` 返回 true，则将项添加到集合中并返回 true，
// 否则不执行任何操作并返回 false。
//
// 注意：如果 `item` 为 nil，则不执行任何操作并返回 false。
// 函数 `f` 是在写入锁内执行的。
func (set *Set) AddIfNotExistFuncLock(item interface{}, f func() bool) bool {
	if item == nil {
		return false
	}
	if !set.Contains(item) {
		set.mu.Lock()
		defer set.mu.Unlock()
		if set.data == nil {
			set.data = make(map[interface{}]struct{})
		}
		if f() {
			if _, ok := set.data[item]; !ok {
				set.data[item] = struct{}{}
				return true
			}
		}
	}
	return false
}

// Contains 检查集合是否包含 `item`。
func (set *Set) Contains(item interface{}) bool {
	var ok bool
	set.mu.RLock()
	if set.data != nil {
		_, ok = set.data[item]
	}
	set.mu.RUnlock()
	return ok
}

// Remove 删除集合中的 `item`。
func (set *Set) Remove(item interface{}) {
	set.mu.Lock()
	if set.data != nil {
		delete(set.data, item)
	}
	set.mu.Unlock()
}

// Size 返回集合中的项数。
func (set *Set) Size() int {
	set.mu.RLock()
	l := len(set.data)
	set.mu.RUnlock()
	return l
}

// Clear 删除集合中的所有项。
func (set *Set) Clear() {
	set.mu.Lock()
	set.data = make(map[interface{}]struct{})
	set.mu.Unlock()
}

// Slice 返回集合中的所有项作为切片。
func (set *Set) Slice() []interface{} {
	set.mu.RLock()
	var (
		i   = 0
		ret = make([]interface{}, len(set.data))
	)
	for item := range set.data {
		ret[i] = item
		i++
	}
	set.mu.RUnlock()
	return ret
}

// Join 将集合中的所有项连接为一个字符串，
// 每个项之间使用 `glue` 分隔。
func (set *Set) Join(glue string) string {
	set.mu.RLock()
	defer set.mu.RUnlock()
	if len(set.data) == 0 {
		return ""
	}
	var (
		l      = len(set.data)
		i      = 0
		buffer = bytes.NewBuffer(nil)
	)
	for k := range set.data {
		buffer.WriteString(gconv.String(k))
		if i != l-1 {
			buffer.WriteString(glue)
		}
		i++
	}
	return buffer.String()
}

// String 返回集合中的所有项作为字符串，
// 每个项之间使用 `,` 分隔，
// 并在字符串两端添加 `[` 和 `]`。
func (set *Set) String() string {
	if set == nil {
		return ""
	}
	set.mu.RLock()
	defer set.mu.RUnlock()
	var (
		s      string
		l      = len(set.data)
		i      = 0
		buffer = bytes.NewBuffer(nil)
	)
	buffer.WriteByte('[')
	for k := range set.data {
		s = gconv.String(k)
		if gstr.IsNumeric(s) {
			buffer.WriteString(s)
		} else {
			buffer.WriteString(`"` + gstr.QuoteMeta(s, `"\`) + `"`)
		}
		if i != l-1 {
			buffer.WriteByte(',')
		}
		i++
	}
	buffer.WriteByte(']')
	return buffer.String()
}

// LockFunc 对集合进行写入锁定，
// 并在解锁后调用回调函数 `f`。
func (set *Set) LockFunc(f func(m map[interface{}]struct{})) {
	set.mu.Lock()
	defer set.mu.Unlock()
	f(set.data)
}

// RLockFunc 对集合进行读取锁定，
// 并在解锁后调用回调函数 `f`。
func (set *Set) RLockFunc(f func(m map[interface{}]struct{})) {
	set.mu.RLock()
	defer set.mu.RUnlock()
	f(set.data)
}

// Equal 检查两个集合是否相等。
func (set *Set) Equal(other *Set) bool {
	if set == other {
		return true
	}
	set.mu.RLock()
	defer set.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()
	if len(set.data) != len(other.data) {
		return false
	}
	for key := range set.data {
		if _, ok := other.data[key]; !ok {
			return false
		}
	}
	return true
}

// IsSubsetOf 检查当前集合是否是 `other` 的子集。
func (set *Set) IsSubsetOf(other *Set) bool {
	if set == other {
		return true
	}
	set.mu.RLock()
	defer set.mu.RUnlock()
	other.mu.RLock()
	defer other.mu.RUnlock()
	for key := range set.data {
		if _, ok := other.data[key]; !ok {
			return false
		}
	}
	return true
}

// Union 返回一个新集合，
// 该集合是 `set` 和 `others` 的并集。
// 这意味着，`newSet` 中的所有项都在 `set` 或 `others` 中。
func (set *Set) Union(others ...*Set) (newSet *Set) {
	newSet = NewSet()
	set.mu.RLock()
	defer set.mu.RUnlock()
	for _, other := range others {
		if set != other {
			other.mu.RLock()
		}
		for k, v := range set.data {
			newSet.data[k] = v
		}
		if set != other {
			for k, v := range other.data {
				newSet.data[k] = v
			}
		}
		if set != other {
			other.mu.RUnlock()
		}
	}

	return
}

// Diff 返回一个新集合，
// 该集合是 `set` 相对于 `others` 的差集。
// 这意味着，`newSet` 中的所有项都在 `set` 中，但不在 `others` 中。
func (set *Set) Diff(others ...*Set) (newSet *Set) {
	newSet = NewSet()
	set.mu.RLock()
	defer set.mu.RUnlock()
	for _, other := range others {
		if set == other {
			continue
		}
		other.mu.RLock()
		for k, v := range set.data {
			if _, ok := other.data[k]; !ok {
				newSet.data[k] = v
			}
		}
		other.mu.RUnlock()
	}
	return
}

// Intersect 返回一个新集合，
// 该集合是 `set` 和 `others` 的交集。
// 这意味着，`newSet` 中的所有项都在 `set` 和 `others` 中。
func (set *Set) Intersect(others ...*Set) (newSet *Set) {
	newSet = NewSet()
	set.mu.RLock()
	defer set.mu.RUnlock()
	for _, other := range others {
		if set != other {
			other.mu.RLock()
		}
		for k, v := range set.data {
			if _, ok := other.data[k]; ok {
				newSet.data[k] = v
			}
		}
		if set != other {
			other.mu.RUnlock()
		}
	}
	return
}

// Complement 返回一个新集合，
// 该集合是 `set` 相对于 `full` 的补集。
// 这意味着，`newSet` 中的所有项都在 `full` 中，但不在 `set` 中。
//
// 如果给定的集合 `full` 不是 `set` 的全集，
// 则返回 `full` 与 `set` 的差集。
func (set *Set) Complement(full *Set) (newSet *Set) {
	newSet = NewSet()
	set.mu.RLock()
	defer set.mu.RUnlock()
	if set != full {
		full.mu.RLock()
		defer full.mu.RUnlock()
	}
	for k, v := range full.data {
		if _, ok := set.data[k]; !ok {
			newSet.data[k] = v
		}
	}
	return
}

// Merge 将 `others` 集合中的所有项添加到 `set` 中。
func (set *Set) Merge(others ...*Set) *Set {
	set.mu.Lock()
	defer set.mu.Unlock()
	for _, other := range others {
		if set != other {
			other.mu.RLock()
		}
		for k, v := range other.data {
			set.data[k] = v
		}
		if set != other {
			other.mu.RUnlock()
		}
	}
	return set
}

// Sum sums items.
// 注意：这些应转换为int类型,
// 要么你会得到意想不到的结果.
func (set *Set) Sum() (sum int) {
	set.mu.RLock()
	defer set.mu.RUnlock()
	for k := range set.data {
		sum += gconv.Int(k)
	}
	return
}

// Pop 随机弹出 `set` 中的一个项。
func (set *Set) Pop() interface{} {
	set.mu.Lock()
	defer set.mu.Unlock()
	for k := range set.data {
		delete(set.data, k)
		return k
	}
	return nil
}

// Pops 随机弹出 `set` 中的 `size` 个项。
// 如果 `size` 为 -1，则返回所有项。
func (set *Set) Pops(size int) []interface{} {
	set.mu.Lock()
	defer set.mu.Unlock()
	if size > len(set.data) || size == -1 {
		size = len(set.data)
	}
	if size <= 0 {
		return nil
	}
	index := 0
	array := make([]interface{}, size)
	for k := range set.data {
		delete(set.data, k)
		array[index] = k
		index++
		if index == size {
			break
		}
	}
	return array
}

// Walk 遍历 `set` 中的所有项，
// 并对每个项应用用户提供的函数 `f`。
func (set *Set) Walk(f func(item interface{}) interface{}) *Set {
	set.mu.Lock()
	defer set.mu.Unlock()
	m := make(map[interface{}]struct{}, len(set.data))
	for k, v := range set.data {
		m[f(k)] = v
	}
	set.data = m
	return set
}

// MarshalJSON implements the interface MarshalJSON for json.Marshal.
func (set Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(set.Slice())
}

// UnmarshalJSON implements the interface UnmarshalJSON for json.Unmarshal.
func (set *Set) UnmarshalJSON(b []byte) error {
	set.mu.Lock()
	defer set.mu.Unlock()
	if set.data == nil {
		set.data = make(map[interface{}]struct{})
	}
	var array []interface{}
	if err := json.UnmarshalUseNumber(b, &array); err != nil {
		return err
	}
	for _, v := range array {
		set.data[v] = struct{}{}
	}
	return nil
}

// UnmarshalValue 是一个接口实现，可以为 set 设置任意类型的值。
func (set *Set) UnmarshalValue(value interface{}) (err error) {
	set.mu.Lock()
	defer set.mu.Unlock()
	if set.data == nil {
		set.data = make(map[interface{}]struct{})
	}
	var array []interface{}
	switch value.(type) {
	case string, []byte:
		err = json.UnmarshalUseNumber(gconv.Bytes(value), &array)
	default:
		array = gconv.SliceAny(value)
	}
	for _, v := range array {
		set.data[v] = struct{}{}
	}
	return
}

// DeepCopy 实现了深拷贝当前类型的接口。
func (set *Set) DeepCopy() interface{} {
	if set == nil {
		return nil
	}
	set.mu.RLock()
	defer set.mu.RUnlock()
	data := make([]interface{}, 0)
	for k := range set.data {
		data = append(data, k)
	}
	return NewFrom(data, set.mu.IsSafe())
}
