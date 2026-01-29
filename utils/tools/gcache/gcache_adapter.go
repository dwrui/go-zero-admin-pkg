package gcache

import (
	"context"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gvar"
	"time"
)

// Adapter 是缓存功能实现的核心适配器接口。
//
// 注意：实现者本身应保证这些函数的并发安全性。
type Adapter interface {
	// Set 使用 `key`-`value` 对设置缓存，在 `duration` 时间后过期。
	//
	// 如果 `duration` == 0，则永不过期。
	// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `data` 的键。
	Set(ctx context.Context, key interface{}, value interface{}, duration time.Duration) error

	// SetMap 批量设置缓存，使用 `data` 映射中的键值对，在 `duration` 时间后过期。
	//
	// 如果 `duration` == 0，则永不过期。
	// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `data` 的键。
	SetMap(ctx context.Context, data map[interface{}]interface{}, duration time.Duration) error

	// SetIfNotExist 仅在 `key` 不存在于缓存中时，使用 `key`-`value` 对设置缓存，在 `duration` 时间后过期。
	// 如果 `key` 不存在于缓存中，返回 true 并成功设置 `value`，否则返回 false。
	//
	// 如果 `duration` == 0，则永不过期。
	// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `key`。
	SetIfNotExist(ctx context.Context, key interface{}, value interface{}, duration time.Duration) (ok bool, err error)

	// SetIfNotExistFunc 仅在 `key` 不存在于缓存中时，使用函数 `f` 的结果设置 `key`，并返回 true；
	// 如果 `key` 已存在，则不做任何操作并返回 false。
	//
	// 参数 `value` 可以是 `func() interface{}` 类型，但如果其结果为 nil，则不做任何操作。
	//
	// 如果 `duration` == 0，则永不过期。
	// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `key`。
	SetIfNotExistFunc(ctx context.Context, key interface{}, f Func, duration time.Duration) (ok bool, err error)

	// SetIfNotExistFuncLock 仅在 `key` 不存在于缓存中时，使用函数 `f` 的结果设置 `key`，并返回 true；
	// 如果 `key` 已存在，则不做任何操作并返回 false。
	//
	// 如果 `duration` == 0，则永不过期。
	// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `key`。
	//
	// 注意：与函数 `SetIfNotExistFunc` 的不同之处在于，函数 `f` 在写锁内执行，以保证并发安全。
	SetIfNotExistFuncLock(ctx context.Context, key interface{}, f Func, duration time.Duration) (ok bool, err error)

	// Get 检索并返回给定 `key` 的关联值。
	// 如果键不存在、值为 nil 或已过期，则返回 nil。
	// 如果你想检查 `key` 是否存在于缓存中，最好使用函数 Contains。
	Get(ctx context.Context, key interface{}) (*gvar.Var, error)

	// GetOrSet 检索并返回 `key` 的值，如果 `key` 不存在于缓存中，则设置 `key`-`value` 对并返回 `value`。
	// 键值对在 `duration` 时间后过期。
	//
	// 如果 `duration` == 0，则永不过期。
	// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `key`，但如果 `value` 是函数且函数结果为 nil，则不做任何操作。
	GetOrSet(ctx context.Context, key interface{}, value interface{}, duration time.Duration) (result *gvar.Var, err error)

	// GetOrSetFunc 检索并返回 `key` 的值，如果 `key` 不存在于缓存中，则使用函数 `f` 的结果设置 `key` 并返回其结果。
	// 键值对在 `duration` 时间后过期。
	//
	// 如果 `duration` == 0，则永不过期。
	// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `key`，但如果 `value` 是函数且函数结果为 nil，则不做任何操作。
	GetOrSetFunc(ctx context.Context, key interface{}, f Func, duration time.Duration) (result *gvar.Var, err error)

	// GetOrSetFuncLock 检索并返回 `key` 的值，如果 `key` 不存在于缓存中，则使用函数 `f` 的结果设置 `key` 并返回其结果。
	// 键值对在 `duration` 时间后过期。
	//
	// 如果 `duration` == 0，则永不过期。
	// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `key`，但如果 `value` 是函数且函数结果为 nil，则不做任何操作。
	//
	// 注意：与函数 `GetOrSetFunc` 的不同之处在于，函数 `f` 在写锁内执行，以保证并发安全。
	GetOrSetFuncLock(ctx context.Context, key interface{}, f Func, duration time.Duration) (result *gvar.Var, err error)

	// Contains 检查并返回 true 如果 `key` 存在于缓存中，否则返回 false。
	Contains(ctx context.Context, key interface{}) (bool, error)

	// Size 返回缓存中的项目数量。
	Size(ctx context.Context) (size int, err error)

	// Data 以映射类型返回缓存中所有键值对的副本。
	// 注意：此函数可能导致大量内存使用，如有必要可实现此函数。
	Data(ctx context.Context) (data map[interface{}]interface{}, err error)

	// Keys 以切片形式返回缓存中的所有键。
	Keys(ctx context.Context) (keys []interface{}, err error)

	// Values 以切片形式返回缓存中的所有值。
	Values(ctx context.Context) (values []interface{}, err error)

	// Update 更新 `key` 的值而不改变其过期时间，并返回旧值。
	// 如果 `key` 不存在于缓存中，返回的值 `exist` 为 false。
	//
	// 如果给定的 `value` 为 nil，则删除 `key`。
	// 如果 `key` 不存在于缓存中，则不做任何操作。
	Update(ctx context.Context, key interface{}, value interface{}) (oldValue *gvar.Var, exist bool, err error)

	// UpdateExpire 更新 `key` 的过期时间，并返回旧的过期时间值。
	//
	// 如果 `key` 不存在于缓存中，返回 -1 且不做任何操作。
	// 如果 `duration` < 0，则删除 `key`。
	UpdateExpire(ctx context.Context, key interface{}, duration time.Duration) (oldDuration time.Duration, err error)

	// GetExpire 检索并返回缓存中 `key` 的过期时间。
	//
	// 注意：
	// 如果 `key` 永不过期，返回 0。
	// 如果 `key` 不存在于缓存中，返回 -1。
	GetExpire(ctx context.Context, key interface{}) (time.Duration, error)

	// Remove 从缓存中删除一个或多个键，并返回其值。
	// 如果给定多个键，返回最后一个被删除项的值。
	Remove(ctx context.Context, keys ...interface{}) (lastValue *gvar.Var, err error)

	// Clear 清除缓存中的所有数据。
	// 注意：此函数较敏感，应谨慎使用。
	Clear(ctx context.Context) error

	// Close 如有必要，关闭缓存。
	Close(ctx context.Context) error
}
