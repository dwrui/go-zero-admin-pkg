// 包 gcache 为进程提供各种缓存管理功能。
// 它默认情况下为进程提供了一个并发安全的内存缓存适配器
package gcache

import (
	"context"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gvar"
	"time"
)

// Func 是缓存函数，用于计算并返回值。
type Func = func(ctx context.Context) (value interface{}, err error)

// DurationNoExpire 表示缓存键值对永不过期。
const DurationNoExpire = time.Duration(0)

// DefaultCache 是默认的缓存对象。
var defaultCache = New()

// Set 方法用于设置缓存，将 `key`-`value` 对存储到缓存中，过期时间为 `duration`。
//
// 如果 `duration` 为 0，则表示永不过期。
// 如果 `duration` 小于 0 或 `value` 为 nil，则会删除对应的缓存键。
func Set(ctx context.Context, key interface{}, value interface{}, duration time.Duration) error {
	return defaultCache.Set(ctx, key, value, duration)
}

// SetMap 方法用于批量设置缓存，将 `data` 中的键值对存储到缓存中，过期时间为 `duration`。
//
// 如果 `duration` 为 0，则表示永不过期。
// 如果 `duration` 小于 0 或 `value` 为 nil，则会删除对应的缓存键。
func SetMap(ctx context.Context, data map[interface{}]interface{}, duration time.Duration) error {
	return defaultCache.SetMap(ctx, data, duration)
}

// SetIfNotExist 方法用于设置缓存，将 `key`-`value` 对存储到缓存中，过期时间为 `duration`，
// 只有当 `key` 不存在于缓存中时才会设置成功。
//
// 如果 `duration` 为 0，则表示永不过期。
// 如果 `duration` 小于 0 或 `value` 为 nil，则会删除对应的缓存键。
// It deletes the `key` if `duration` < 0 or given `value` is nil.
func SetIfNotExist(ctx context.Context, key interface{}, value interface{}, duration time.Duration) (bool, error) {
	return defaultCache.SetIfNotExist(ctx, key, value, duration)
}

// SetIfNotExistFunc 方法用于设置缓存，将 `key` 与函数 `f` 的结果存储到缓存中，过期时间为 `duration`，
// 只有当 `key` 不存在于缓存中时才会设置成功。
//
// 如果 `duration` 为 0，则表示永不过期。
// 如果 `duration` 小于 0 或 `value` 为 nil，则会删除对应的缓存键。
//
// The parameter `f` must be a function that returns a value and an error.
// If the result value is nil, it will not be set to the cache.
//
// 如果 `duration` 为 0，则表示永不过期。
// 如果 `duration` 小于 0 或 `value` 为 nil，则会删除对应的缓存键。
func SetIfNotExistFunc(ctx context.Context, key interface{}, f Func, duration time.Duration) (bool, error) {
	return defaultCache.SetIfNotExistFunc(ctx, key, f, duration)
}

// SetIfNotExistFuncLock 使用函数 `f` 的结果设置 `key`，并返回 true
// 如果缓存中不存在`key`，否则什么都不做；如果`key`已存在，则返回false。
//
// 如果 `duration` 为 0，则表示永不过期。
// 如果 `duration` 小于 0 或 `value` 为 nil，则会删除对应的缓存键。
//
// 注意，它与`SetIfNotExistFunc`函数的不同之处在于，`f`函数是在内部执行的
// 为了并发安全，正在写入互斥锁。
func SetIfNotExistFuncLock(ctx context.Context, key interface{}, f Func, duration time.Duration) (bool, error) {
	return defaultCache.SetIfNotExistFuncLock(ctx, key, f, duration)
}

// Get 函数用于检索并返回给定 `key` 的关联值。
// 如果它不存在、其值为 nil 或者已过期，则返回 nil。
// 如果您想检查缓存中是否存在`key`，最好使用Contains函数。
func Get(ctx context.Context, key interface{}) (*gvar.Var, error) {
	return defaultCache.Get(ctx, key)
}

// GetOrSet 检索并返回 `key` 的值，或者设置 `key`-`value` 对
// 如果缓存中不存在`key`，则返回`value`。键值对过期
// 在 `duration` 之后。
//
// 如果 `duration` == 0，则不会过期。
// 如果`duration`小于0或给定的`value`为空，则删除`key`，否则不执行任何操作

func GetOrSet(ctx context.Context, key interface{}, value interface{}, duration time.Duration) (*gvar.Var, error) {
	return defaultCache.GetOrSet(ctx, key, value, duration)
}

// GetOrSetFunc 检索并返回 `key` 的值，或者用结果设置 `key`
// 函数 `f`，如果缓存中不存在 `key`，则返回其结果。键值
// 配对在`duration`后过期。
//
// 如果 `duration` == 0，则不会过期。
// 如果`duration`小于0或者给定的`value`为空，则删除`key`，否则不执行任何操作
// 如果 `value` 是一个函数，且该函数的结果为 nil。
func GetOrSetFunc(ctx context.Context, key interface{}, f Func, duration time.Duration) (*gvar.Var, error) {
	return defaultCache.GetOrSetFunc(ctx, key, f, duration)
}

// GetOrSetFuncLock 检索并返回 `key` 的值，或者用结果设置 `key`
// 函数 `f`，如果缓存中不存在 `key`，则返回其结果。键值
// 配对在`duration`后过期。
//
// 如果 `duration` == 0，则不会过期。
// 如果`duration`小于0或给定的`value`为空，则删除`key`，否则不执行任何操作
// 如果`value`是一个函数，且该函数的结果为空。
//
// 注意，它与`GetOrSetFunc`函数的不同之处在于，`f`函数是在内部执行的
// 为并发安全目的编写互斥锁。
func GetOrSetFuncLock(ctx context.Context, key interface{}, f Func, duration time.Duration) (*gvar.Var, error) {
	return defaultCache.GetOrSetFuncLock(ctx, key, f, duration)
}

// 包含检查，如果`key`存在于缓存中，则返回true，否则返回false。
func Contains(ctx context.Context, key interface{}) (bool, error) {
	return defaultCache.Contains(ctx, key)
}

// GetExpire 检索并返回 `key` 在缓存中的过期时间。
//
// 注意，
// 如果 `key` 永不过期，则返回 0。
// 如果 `key` 不存在于缓存中，则返回 -1。
func GetExpire(ctx context.Context, key interface{}) (time.Duration, error) {
	return defaultCache.GetExpire(ctx, key)
}

// Remove 删除缓存中的一个或多个键，并返回其值。
// 如果给定多个键，则返回最后删除项的值。
func Remove(ctx context.Context, keys ...interface{}) (value *gvar.Var, err error) {
	return defaultCache.Remove(ctx, keys...)
}

// Removes deletes `keys` in the cache.
func Removes(ctx context.Context, keys []interface{}) error {
	return defaultCache.Removes(ctx, keys)
}

// `Update` 函数用于更新 `key` 的值，但不改变其过期时间，并返回旧值。
// 如果缓存中不存在`key`，则返回值`exist`为false。
//
// 如果给定的 `value` 为 nil，则删除 `key`。
// 如果缓存中不存在`key`，则不执行任何操作
func Update(ctx context.Context, key interface{}, value interface{}) (oldValue *gvar.Var, exist bool, err error) {
	return defaultCache.Update(ctx, key, value)
}

// UpdateExpire 更新 `key` 的过期时间，并返回旧的过期时间值。
//
// 如果 `key` 不存在于缓存中，则返回值 `oldDuration` 为 -1。
// 如果 `duration` < 0，则删除 `key`。
func UpdateExpire(ctx context.Context, key interface{}, duration time.Duration) (oldDuration time.Duration, err error) {
	return defaultCache.UpdateExpire(ctx, key, duration)
}

// Size 返回缓存中的项目数
func Size(ctx context.Context) (int, error) {
	return defaultCache.Size(ctx)
}

// Data 以 map 类型返回缓存中所有键值对的副本。
// 注意，此函数可能会占用大量内存，您可以实现此函数
// 如有必要
func Data(ctx context.Context) (map[interface{}]interface{}, error) {
	return defaultCache.Data(ctx)
}

// 键返回缓存中的所有键作为切片。
func Keys(ctx context.Context) ([]interface{}, error) {
	return defaultCache.Keys(ctx)
}

// KeyStrings 返回缓存中的所有键作为字符串切片。
func KeyStrings(ctx context.Context) ([]string, error) {
	return defaultCache.KeyStrings(ctx)
}

// Values 以切片形式返回缓存中的所有值。
func Values(ctx context.Context) ([]interface{}, error) {
	return defaultCache.Values(ctx)
}

// MustGet 表现得像 Get，但一旦出现错误就会恐慌。
func MustGet(ctx context.Context, key interface{}) *gvar.Var {
	return defaultCache.MustGet(ctx, key)
}

// MustGetOrSet 表现得像 GetOrSet，但一旦出现错误就会恐慌。
func MustGetOrSet(ctx context.Context, key interface{}, value interface{}, duration time.Duration) *gvar.Var {
	return defaultCache.MustGetOrSet(ctx, key, value, duration)
}

// MustGetOrSetFunc 表现得像 GetOrSetFunc，但一旦出现错误就会恐慌。
func MustGetOrSetFunc(ctx context.Context, key interface{}, f Func, duration time.Duration) *gvar.Var {
	return defaultCache.MustGetOrSetFunc(ctx, key, f, duration)
}

// MustGetOrSetFuncLock 的行为类似于 GetOrSetFuncLock，但如果出现错误它会慌乱。
func MustGetOrSetFuncLock(ctx context.Context, key interface{}, f Func, duration time.Duration) *gvar.Var {
	return defaultCache.MustGetOrSetFuncLock(ctx, key, f, duration)
}

// MustContains 的表现类似于 Contains，但如果出现错误它会慌乱。
func MustContains(ctx context.Context, key interface{}) bool {
	return defaultCache.MustContains(ctx, key)
}

// MustGetExpire 的表现就像 GetExpire，但如果出现错误它会慌乱。
func MustGetExpire(ctx context.Context, key interface{}) time.Duration {
	return defaultCache.MustGetExpire(ctx, key)
}

// MustSize 就像 Size 一样，但如果出现错误它会慌乱。
func MustSize(ctx context.Context) int {
	return defaultCache.MustSize(ctx)
}

// MustData 的行为像 Data，但一旦出现错误就会慌乱。
func MustData(ctx context.Context) map[interface{}]interface{} {
	return defaultCache.MustData(ctx)
}

// MustKeys 就像 Keys，但如果出现错误就会慌乱。
func MustKeys(ctx context.Context) []interface{} {
	return defaultCache.MustKeys(ctx)
}

// MustKeyStrings 像 KeyStrings 一样，但如果出现错误就会慌乱。
func MustKeyStrings(ctx context.Context) []string {
	return defaultCache.MustKeyStrings(ctx)
}

// MustValues 就像 Values，但如果出现错误它会慌乱。
func MustValues(ctx context.Context) []interface{} {
	return defaultCache.MustValues(ctx)
}
