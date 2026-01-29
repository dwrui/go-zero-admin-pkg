//

package gset

import (
	"bytes"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/json"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/rwmutex"
)

// IntSet 由 int 项组成的集合。
type IntSet struct {
	mu   rwmutex.RWMutex
	data map[int]struct{}
}

// NewIntSet 创建并返回一个新的集合，其中包含不重复的 int 项。
// 参数 `safe` 用于指定是否在并发安全中使用集合，默认情况下是 false。
func NewIntSet(safe ...bool) *IntSet {
	return &IntSet{
		mu:   rwmutex.Create(safe...),
		data: make(map[int]struct{}),
	}
}

// NewIntSetFrom 返回一个新的集合，其中包含 `items` 中的不重复项。
// 参数 `safe` 用于指定是否在并发安全中使用集合，默认情况下是 false。
func NewIntSetFrom(items []int, safe ...bool) *IntSet {
	m := make(map[int]struct{})
	for _, v := range items {
		m[v] = struct{}{}
	}
	return &IntSet{
		mu:   rwmutex.Create(safe...),
		data: m,
	}
}

// Iterator 遍历集合中的所有项，只读模式。
// 它使用给定的回调函数 `f` 对每个项进行迭代，
// 如果“f”返回为真，则继续迭代;或false停止迭代.
func (set *IntSet) Iterator(f func(v int) bool) {
	for _, k := range set.Slice() {
		if !f(k) {
			break
		}
	}
}

// Add 添加一个或多个项到集合中。
func (set *IntSet) Add(item ...int) {
	set.mu.Lock()
	if set.data == nil {
		set.data = make(map[int]struct{})
	}
	for _, v := range item {
		set.data[v] = struct{}{}
	}
	set.mu.Unlock()
}

// AddIfNotExist 检查集合中是否存在 `item`，
// 如果不存在，则将 `item` 添加到集合中并返回 true；
// 否则，不执行任何操作并返回 false。
//
// 注意：如果 `item` 为 nil，则不执行任何操作并返回 false。
func (set *IntSet) AddIfNotExist(item int) bool {
	if !set.Contains(item) {
		set.mu.Lock()
		defer set.mu.Unlock()
		if set.data == nil {
			set.data = make(map[int]struct{})
		}
		if _, ok := set.data[item]; !ok {
			set.data[item] = struct{}{}
			return true
		}
	}
	return false
}

// AddIfNotExistFunc 检查集合中是否存在 `item`，
// 如果不存在，则将 `item` 添加到集合中并返回 true；
// 否则，不执行任何操作并返回 false。
//
// 注意：
// 1. 如果 `item` 为 nil，则不执行任何操作并返回 false。
// 2. 函数 `f` 在没有写入锁的情况下执行。
func (set *IntSet) AddIfNotExistFunc(item int, f func() bool) bool {
	if !set.Contains(item) {
		if f() {
			set.mu.Lock()
			defer set.mu.Unlock()
			if set.data == nil {
				set.data = make(map[int]struct{})
			}
			if _, ok := set.data[item]; !ok {
				set.data[item] = struct{}{}
				return true
			}
		}
	}
	return false
}

// AddIfNotExistFuncLock 检查集合中是否存在 `item`，
// 如果不存在，则将 `item` 添加到集合中并返回 true；
// 否则，不执行任何操作并返回 false。
//
// 注意：
// 1. 如果 `item` 为 nil，则不执行任何操作并返回 false。
// 2. 函数 `f` 在有写入锁的情况下执行。
func (set *IntSet) AddIfNotExistFuncLock(item int, f func() bool) bool {
	if !set.Contains(item) {
		set.mu.Lock()
		defer set.mu.Unlock()
		if set.data == nil {
			set.data = make(map[int]struct{})
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

// Contains 检查集合中是否存在 `item`。
func (set *IntSet) Contains(item int) bool {
	var ok bool
	set.mu.RLock()
	if set.data != nil {
		_, ok = set.data[item]
	}
	set.mu.RUnlock()
	return ok
}

// Remove 删除集合中的 `item`。
func (set *IntSet) Remove(item int) {
	set.mu.Lock()
	if set.data != nil {
		delete(set.data, item)
	}
	set.mu.Unlock()
}

// Size 返回集合中项的数量。
func (set *IntSet) Size() int {
	set.mu.RLock()
	l := len(set.data)
	set.mu.RUnlock()
	return l
}

// Clear 删除集合中的所有项。
func (set *IntSet) Clear() {
	set.mu.Lock()
	set.data = make(map[int]struct{})
	set.mu.Unlock()
}

// Slice 返回集合中项的切片表示。
func (set *IntSet) Slice() []int {
	set.mu.RLock()
	var (
		i   = 0
		ret = make([]int, len(set.data))
	)
	for k := range set.data {
		ret[i] = k
		i++
	}
	set.mu.RUnlock()
	return ret
}

// Join 将集合中的项连接为一个字符串，使用 `glue` 作为分隔符。
func (set *IntSet) Join(glue string) string {
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

// String 返回集合中项的字符串表示，实现了与 json.Marshal 相同的行为。
func (set *IntSet) String() string {
	if set == nil {
		return ""
	}
	return "[" + set.Join(",") + "]"
}

// LockFunc 用回调函数 'f' 锁写入.
func (set *IntSet) LockFunc(f func(m map[int]struct{})) {
	set.mu.Lock()
	defer set.mu.Unlock()
	f(set.data)
}

// RLockFunc 通过回调函数 'f' 锁定读取.
func (set *IntSet) RLockFunc(f func(m map[int]struct{})) {
	set.mu.RLock()
	defer set.mu.RUnlock()
	f(set.data)
}

// Equal 检查两个集合是否相等。
func (set *IntSet) Equal(other *IntSet) bool {
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

// IsSubsetOf 检查当前集合是否为 `other` 的子集。
func (set *IntSet) IsSubsetOf(other *IntSet) bool {
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

// Union 返回一个新集合，该集合是 `set` 和 `other` 的并集。
// 这意味着，`newSet` 中的所有项都在 `set` 或 `other` 中。
func (set *IntSet) Union(others ...*IntSet) (newSet *IntSet) {
	newSet = NewIntSet()
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

// Diff 返回一个新集合，该集合是 `set` 到 `other` 的差集。
// 这意味着，`newSet` 中的所有项都在 `set` 中，但不在 `other` 中。
func (set *IntSet) Diff(others ...*IntSet) (newSet *IntSet) {
	newSet = NewIntSet()
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

// Intersect 返回一个新集合，该集合是 `set` 和 `other` 的交集。
// 这意味着，`newSet` 中的所有项都在 `set` 和 `other` 中。
func (set *IntSet) Intersect(others ...*IntSet) (newSet *IntSet) {
	newSet = NewIntSet()
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

// Complement 返回一个新集合，该集合是 `set` 到 `full` 的补集。
// 这意味着，`newSet` 中的所有项都在 `full` 中，但不在 `set` 中。
//
// 如果给定的集合 `full` 不是 `set` 的完整集合，则返回 `full` 到 `set` 的差集。
func (set *IntSet) Complement(full *IntSet) (newSet *IntSet) {
	newSet = NewIntSet()
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

// Merge 将 `others` 集合中的项添加到 `set` 中。
func (set *IntSet) Merge(others ...*IntSet) *IntSet {
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

// Sum 返回集合中所有项的总和。
// 注意：集合中的项应转换为 int 类型，
// 否则您可能会得到一个意外的结果。
func (set *IntSet) Sum() (sum int) {
	set.mu.RLock()
	defer set.mu.RUnlock()
	for k := range set.data {
		sum += k
	}
	return
}

// Pop 随机弹出集合中的一个项。
func (set *IntSet) Pop() int {
	set.mu.Lock()
	defer set.mu.Unlock()
	for k := range set.data {
		delete(set.data, k)
		return k
	}
	return 0
}

// Pops 随机弹出集合中的 `size` 个项。
// 如果 `size` 为 -1，则返回所有项。
func (set *IntSet) Pops(size int) []int {
	set.mu.Lock()
	defer set.mu.Unlock()
	if size > len(set.data) || size == -1 {
		size = len(set.data)
	}
	if size <= 0 {
		return nil
	}
	index := 0
	array := make([]int, size)
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

// Walk 遍历集合中的所有项，并应用用户提供的函数 `f` 到每个项。
func (set *IntSet) Walk(f func(item int) int) *IntSet {
	set.mu.Lock()
	defer set.mu.Unlock()
	m := make(map[int]struct{}, len(set.data))
	for k, v := range set.data {
		m[f(k)] = v
	}
	set.data = m
	return set
}

// MarshalJSON implements the interface MarshalJSON for json.Marshal.
func (set IntSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(set.Slice())
}

// UnmarshalJSON implements the interface UnmarshalJSON for json.Unmarshal.
func (set *IntSet) UnmarshalJSON(b []byte) error {
	set.mu.Lock()
	defer set.mu.Unlock()
	if set.data == nil {
		set.data = make(map[int]struct{})
	}
	var array []int
	if err := json.UnmarshalUseNumber(b, &array); err != nil {
		return err
	}
	for _, v := range array {
		set.data[v] = struct{}{}
	}
	return nil
}

// UnmarshalJSON 实现 json.Unmarshal 的 UnmarshalJSON 接口。
func (set *IntSet) UnmarshalValue(value interface{}) (err error) {
	set.mu.Lock()
	defer set.mu.Unlock()
	if set.data == nil {
		set.data = make(map[int]struct{})
	}
	var array []int
	switch value.(type) {
	case string, []byte:
		err = json.UnmarshalUseNumber(gconv.Bytes(value), &array)
	default:
		array = gconv.SliceInt(value)
	}
	for _, v := range array {
		set.data[v] = struct{}{}
	}
	return
}

// DeepCopy implements interface for deep copy of current type.
func (set *IntSet) DeepCopy() interface{} {
	if set == nil {
		return nil
	}
	set.mu.RLock()
	defer set.mu.RUnlock()
	var (
		slice = make([]int, len(set.data))
		index = 0
	)
	for k := range set.data {
		slice[index] = k
		index++
	}
	return NewIntSetFrom(slice, set.mu.IsSafe())
}
