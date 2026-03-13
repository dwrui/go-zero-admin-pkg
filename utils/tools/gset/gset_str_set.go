//

package gset

import (
	"bytes"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gstr"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/json"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/rwmutex"
	"strings"
)

// StrSet 是一个字符串集合，其中包含不重复的项。
type StrSet struct {
	mu   rwmutex.RWMutex
	data map[string]struct{}
}

// NewStrSet 创建并返回一个新的集合，该集合包含不重复的项。
// 参数 `safe` 用于指定是否在并发安全模式下使用集合，默认值为 false。
func NewStrSet(safe ...bool) *StrSet {
	return &StrSet{
		mu:   rwmutex.Create(safe...),
		data: make(map[string]struct{}),
	}
}

// NewStrSetFrom 创建并返回一个新的集合，该集合包含 `items` 中的不重复项。
// 参数 `safe` 用于指定是否在并发安全模式下使用集合，默认值为 false。
func NewStrSetFrom(items []string, safe ...bool) *StrSet {
	m := make(map[string]struct{})
	for _, v := range items {
		m[v] = struct{}{}
	}
	return &StrSet{
		mu:   rwmutex.Create(safe...),
		data: m,
	}
}

// Iterator 遍历集合中的所有项，并使用给定的回调函数 `f` 对每个项进行处理。
// 如果 `f` 返回 true，则继续迭代；否则停止迭代。
func (set *StrSet) Iterator(f func(v string) bool) {
	for _, k := range set.Slice() {
		if !f(k) {
			break
		}
	}
}

// Add 添加一个或多个项到集合中。
func (set *StrSet) Add(item ...string) {
	set.mu.Lock()
	if set.data == nil {
		set.data = make(map[string]struct{})
	}
	for _, v := range item {
		set.data[v] = struct{}{}
	}
	set.mu.Unlock()
}

// AddIfNotExist 检查集合中是否存在 `item`，如果不存在，则将其添加到集合中并返回 true；
// 如果存在，则不执行任何操作并返回 false。
func (set *StrSet) AddIfNotExist(item string) bool {
	if !set.Contains(item) {
		set.mu.Lock()
		defer set.mu.Unlock()
		if set.data == nil {
			set.data = make(map[string]struct{})
		}
		if _, ok := set.data[item]; !ok {
			set.data[item] = struct{}{}
			return true
		}
	}
	return false
}

// AddIfNotExistFunc 检查集合中是否存在 `item`，如果不存在，则将其添加到集合中并返回 true；
// 如果存在，则不执行任何操作并返回 false。
//
// 注意：函数 `f` 在没有写入锁的情况下执行。
func (set *StrSet) AddIfNotExistFunc(item string, f func() bool) bool {
	if !set.Contains(item) {
		if f() {
			set.mu.Lock()
			defer set.mu.Unlock()
			if set.data == nil {
				set.data = make(map[string]struct{})
			}
			if _, ok := set.data[item]; !ok {
				set.data[item] = struct{}{}
				return true
			}
		}
	}
	return false
}

// AddIfNotExistFuncLock 检查集合中是否存在 `item`，如果不存在，则将其添加到集合中并返回 true；
// 如果存在，则不执行任何操作并返回 false。
//
// 注意：函数 `f` 在有写入锁的情况下执行。
func (set *StrSet) AddIfNotExistFuncLock(item string, f func() bool) bool {
	if !set.Contains(item) {
		set.mu.Lock()
		defer set.mu.Unlock()
		if set.data == nil {
			set.data = make(map[string]struct{})
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
func (set *StrSet) Contains(item string) bool {
	var ok bool
	set.mu.RLock()
	if set.data != nil {
		_, ok = set.data[item]
	}
	set.mu.RUnlock()
	return ok
}

// ContainsI 检查集合是否包含 `item`，并忽略大小写。
// 注意：内部会遍历整个集合来进行大小写不敏感的比较。
func (set *StrSet) ContainsI(item string) bool {
	set.mu.RLock()
	defer set.mu.RUnlock()
	for k := range set.data {
		if strings.EqualFold(k, item) {
			return true
		}
	}
	return false
}

// Remove deletes `item` from set.
func (set *StrSet) Remove(item string) {
	set.mu.Lock()
	if set.data != nil {
		delete(set.data, item)
	}
	set.mu.Unlock()
}

// Size 返回集合中项的数量。
func (set *StrSet) Size() int {
	set.mu.RLock()
	l := len(set.data)
	set.mu.RUnlock()
	return l
}

// Clear 删除集合中的所有项。
func (set *StrSet) Clear() {
	set.mu.Lock()
	set.data = make(map[string]struct{})
	set.mu.Unlock()
}

// Slice 返回集合中的所有项作为切片。
func (set *StrSet) Slice() []string {
	set.mu.RLock()
	var (
		i   = 0
		ret = make([]string, len(set.data))
	)
	for item := range set.data {
		ret[i] = item
		i++
	}

	set.mu.RUnlock()
	return ret
}

// Join 将集合中的项连接为一个字符串，使用 `glue` 作为分隔符。
func (set *StrSet) Join(glue string) string {
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
		buffer.WriteString(k)
		if i != l-1 {
			buffer.WriteString(glue)
		}
		i++
	}
	return buffer.String()
}

// String 返回集合中的所有项作为一个字符串，实现了类似 json.Marshal 方法的行为。
func (set *StrSet) String() string {
	if set == nil {
		return ""
	}
	set.mu.RLock()
	defer set.mu.RUnlock()
	var (
		l      = len(set.data)
		i      = 0
		buffer = bytes.NewBuffer(nil)
	)
	buffer.WriteByte('[')
	for k := range set.data {
		buffer.WriteString(`"` + gstr.QuoteMeta(k, `"\`) + `"`)
		if i != l-1 {
			buffer.WriteByte(',')
		}
		i++
	}
	buffer.WriteByte(']')
	return buffer.String()
}

// LockFunc 用回调函数 'f' 锁写入.
func (set *StrSet) LockFunc(f func(m map[string]struct{})) {
	set.mu.Lock()
	defer set.mu.Unlock()
	f(set.data)
}

// RLockFunc 通过回调函数 'f' 锁定读取.
func (set *StrSet) RLockFunc(f func(m map[string]struct{})) {
	set.mu.RLock()
	defer set.mu.RUnlock()
	f(set.data)
}

// Equal 检查两个集合是否相等。
func (set *StrSet) Equal(other *StrSet) bool {
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
func (set *StrSet) IsSubsetOf(other *StrSet) bool {
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
func (set *StrSet) Union(others ...*StrSet) (newSet *StrSet) {
	newSet = NewStrSet()
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

// Diff 返回一个新集合，该集合是 `set` 相对于 `other` 的差集。
// 这意味着，`newSet` 中的所有项都在 `set` 中，但不在 `other` 中。
func (set *StrSet) Diff(others ...*StrSet) (newSet *StrSet) {
	newSet = NewStrSet()
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
func (set *StrSet) Intersect(others ...*StrSet) (newSet *StrSet) {
	newSet = NewStrSet()
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

// Complement 返回一个新集合，该集合是 `set` 相对于 `full` 的补集。
// 这意味着，`newSet` 中的所有项都在 `full` 中，但不在 `set` 中。
//
// 如果给定的集合 `full` 不是 `set` 的完整集合，则返回 `full` 和 `set` 的差集。
func (set *StrSet) Complement(full *StrSet) (newSet *StrSet) {
	newSet = NewStrSet()
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
func (set *StrSet) Merge(others ...*StrSet) *StrSet {
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

// Sum 将集合中的所有项转换为 int 类型并求和。
// 注意：集合中的项应该能够转换为 int 类型，否则会得到一个意想不到的结果。
func (set *StrSet) Sum() (sum int) {
	set.mu.RLock()
	defer set.mu.RUnlock()
	for k := range set.data {
		sum += gconv.Int(k)
	}
	return
}

// Pop 随机弹出集合中的一个项。
func (set *StrSet) Pop() string {
	set.mu.Lock()
	defer set.mu.Unlock()
	for k := range set.data {
		delete(set.data, k)
		return k
	}
	return ""
}

// Pops 随机弹出集合中的 `size` 个项。
// 如果 `size` 为 -1，则弹出所有项。
func (set *StrSet) Pops(size int) []string {
	set.mu.Lock()
	defer set.mu.Unlock()
	if size > len(set.data) || size == -1 {
		size = len(set.data)
	}
	if size <= 0 {
		return nil
	}
	index := 0
	array := make([]string, size)
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

// Walk 遍历集合中的所有项，并对每个项应用用户提供的函数 `f`。
func (set *StrSet) Walk(f func(item string) string) *StrSet {
	set.mu.Lock()
	defer set.mu.Unlock()
	m := make(map[string]struct{}, len(set.data))
	for k, v := range set.data {
		m[f(k)] = v
	}
	set.data = m
	return set
}

// MarshalJSON implements the interface MarshalJSON for json.Marshal.
func (set StrSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(set.Slice())
}

// UnmarshalJSON implements the interface UnmarshalJSON for json.Unmarshal.
func (set *StrSet) UnmarshalJSON(b []byte) error {
	set.mu.Lock()
	defer set.mu.Unlock()
	if set.data == nil {
		set.data = make(map[string]struct{})
	}
	var array []string
	if err := json.UnmarshalUseNumber(b, &array); err != nil {
		return err
	}
	for _, v := range array {
		set.data[v] = struct{}{}
	}
	return nil
}

// UnmarshalValue is an interface implement which sets any type of value for set.
func (set *StrSet) UnmarshalValue(value interface{}) (err error) {
	set.mu.Lock()
	defer set.mu.Unlock()
	if set.data == nil {
		set.data = make(map[string]struct{})
	}
	var array []string
	switch value.(type) {
	case string, []byte:
		err = json.UnmarshalUseNumber(gconv.Bytes(value), &array)
	default:
		array = gconv.SliceStr(value)
	}
	for _, v := range array {
		set.data[v] = struct{}{}
	}
	return
}

// DeepCopy implements interface for deep copy of current type.
func (set *StrSet) DeepCopy() interface{} {
	if set == nil {
		return nil
	}
	set.mu.RLock()
	defer set.mu.RUnlock()
	var (
		slice = make([]string, len(set.data))
		index = 0
	)
	for k := range set.data {
		slice[index] = k
		index++
	}
	return NewStrSetFrom(slice, set.mu.IsSafe())
}
