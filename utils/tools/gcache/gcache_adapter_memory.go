package gcache

import (
	"context"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/glist"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gset"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gtime"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gtimer"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gtype"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gvar"
	"math"
	"time"
)

// AdapterMemory 是一个使用内存实现的适配器。
type AdapterMemory struct {
	data        *memoryData        // data 是底层缓存数据，存储在哈希表中。
	expireTimes *memoryExpireTimes // expireTimes 是过期键到其时间戳的映射，用于快速索引和删除。
	expireSets  *memoryExpireSets  // expireSets 是过期时间戳到其键集合的映射，用于快速索引和删除。
	lru         *memoryLru         // lru 是 LRU 管理器，当属性 cap > 0 时启用。
	eventList   *glist.List        // eventList 是用于内部数据同步的异步事件列表。
	closed      *gtype.Bool        // closed 控制缓存是否关闭。
}

// 内部事件项。
type adapterMemoryEvent struct {
	k interface{} // 键。
	e int64       // 过期时间（毫秒）。
}

const (
	// defaultMaxExpire 是无过期项的默认过期时间。
	// 它等于 math.MaxInt64/1000000。
	defaultMaxExpire = 9223372036854
)

// NewAdapterMemory 创建并返回一个新的内存适配器缓存对象。
func NewAdapterMemory() *AdapterMemory {
	return doNewAdapterMemory()
}

// NewAdapterMemoryLru 创建并返回一个带 LRU 的新内存适配器缓存对象。
func NewAdapterMemoryLru(cap int) *AdapterMemory {
	c := doNewAdapterMemory()
	c.lru = newMemoryLru(cap)
	return c
}

// doNewAdapterMemory 创建并返回一个新的内存适配器缓存对象。
func doNewAdapterMemory() *AdapterMemory {
	c := &AdapterMemory{
		data:        newMemoryData(),
		expireTimes: newMemoryExpireTimes(),
		expireSets:  newMemoryExpireSets(),
		eventList:   glist.New(true),
		closed:      gtype.NewBool(),
	}
	// 这里如果手动从内存适配器切换适配器，可能会有"定时器泄漏"。
	// 不用担心这个问题，因为适配器很少更改，如果不使用它也不会做任何事情。
	gtimer.AddSingleton(context.Background(), time.Second, c.syncEventAndClearExpired)
	return c
}

// Set 使用 `key`-`value` 对设置缓存，在 `duration` 时间后过期。
//
// 如果 `duration` == 0，则永不过期。
// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `data` 的键。
func (c *AdapterMemory) Set(ctx context.Context, key interface{}, value interface{}, duration time.Duration) error {
	defer c.handleLruKey(ctx, key)
	expireTime := c.getInternalExpire(duration)
	c.data.Set(key, memoryDataItem{
		v: value,
		e: expireTime,
	})
	c.eventList.PushBack(&adapterMemoryEvent{
		k: key,
		e: expireTime,
	})
	return nil
}

// SetMap 批量设置缓存，使用 `data` 映射中的键值对，在 `duration` 时间后过期。
//
// 如果 `duration` == 0，则永不过期。
// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `data` 的键。
func (c *AdapterMemory) SetMap(ctx context.Context, data map[interface{}]interface{}, duration time.Duration) error {
	var (
		expireTime = c.getInternalExpire(duration)
		err        = c.data.SetMap(data, expireTime)
	)
	if err != nil {
		return err
	}
	for k := range data {
		c.eventList.PushBack(&adapterMemoryEvent{
			k: k,
			e: expireTime,
		})
	}
	if c.lru != nil {
		for key := range data {
			c.handleLruKey(ctx, key)
		}
	}
	return nil
}

// SetIfNotExist 仅在 `key` 不存在于缓存中时，使用 `key`-`value` 对设置缓存，在 `duration` 时间后过期。
// 如果 `key` 不存在于缓存中，返回 true 并成功设置 `value`，否则返回 false。
//
// 如果 `duration` == 0，则永不过期。
// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `key`。
func (c *AdapterMemory) SetIfNotExist(ctx context.Context, key interface{}, value interface{}, duration time.Duration) (bool, error) {
	defer c.handleLruKey(ctx, key)
	isContained, err := c.Contains(ctx, key)
	if err != nil {
		return false, err
	}
	if !isContained {
		if _, err = c.doSetWithLockCheck(ctx, key, value, duration); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// SetIfNotExistFunc 仅在 `key` 不存在于缓存中时，使用函数 `f` 的结果设置 `key`，并返回 true；
// 如果 `key` 已存在，则不做任何操作并返回 false。
//
// 参数 `value` 可以是 `func() interface{}` 类型，但如果其结果为 nil，则不做任何操作。
//
// 如果 `duration` == 0，则永不过期。
// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `key`。
func (c *AdapterMemory) SetIfNotExistFunc(ctx context.Context, key interface{}, f Func, duration time.Duration) (bool, error) {
	defer c.handleLruKey(ctx, key)
	isContained, err := c.Contains(ctx, key)
	if err != nil {
		return false, err
	}
	if !isContained {
		value, err := f(ctx)
		if err != nil {
			return false, err
		}
		if _, err = c.doSetWithLockCheck(ctx, key, value, duration); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// SetIfNotExistFuncLock 仅在 `key` 不存在于缓存中时，使用函数 `f` 的结果设置 `key`，并返回 true；
// 如果 `key` 已存在，则不做任何操作并返回 false。
//
// 如果 `duration` == 0，则永不过期。
// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `key`。
//
// 注意：与函数 `SetIfNotExistFunc` 的不同之处在于，函数 `f` 在写锁内执行，以保证并发安全。
func (c *AdapterMemory) SetIfNotExistFuncLock(ctx context.Context, key interface{}, f Func, duration time.Duration) (bool, error) {
	defer c.handleLruKey(ctx, key)
	isContained, err := c.Contains(ctx, key)
	if err != nil {
		return false, err
	}
	if !isContained {
		if _, err = c.doSetWithLockCheck(ctx, key, f, duration); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// Get 检索并返回给定 `key` 的关联值。
// 如果键不存在、值为 nil 或已过期，则返回 nil。
// 如果你想检查 `key` 是否存在于缓存中，最好使用函数 Contains。
func (c *AdapterMemory) Get(ctx context.Context, key interface{}) (*gvar.Var, error) {
	item, ok := c.data.Get(key)
	if ok && !item.IsExpired() {
		c.handleLruKey(ctx, key)
		return gvar.New(item.v), nil
	}
	return nil, nil
}

// GetOrSet 检索并返回 `key` 的值，如果 `key` 不存在于缓存中，则设置 `key`-`value` 对并返回 `value`。
// 键值对在 `duration` 时间后过期。
//
// 如果 `duration` == 0，则永不过期。
// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `key`，但如果 `value` 是函数且函数结果为 nil，则不做任何操作。
func (c *AdapterMemory) GetOrSet(ctx context.Context, key interface{}, value interface{}, duration time.Duration) (*gvar.Var, error) {
	defer c.handleLruKey(ctx, key)
	v, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return c.doSetWithLockCheck(ctx, key, value, duration)
	}
	return v, nil
}

// GetOrSetFunc 检索并返回 `key` 的值，如果 `key` 不存在于缓存中，则使用函数 `f` 的结果设置 `key` 并返回其结果。
// 键值对在 `duration` 时间后过期。
//
// 如果 `duration` == 0，则永不过期。
// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `key`，但如果 `value` 是函数且函数结果为 nil，则不做任何操作。
func (c *AdapterMemory) GetOrSetFunc(ctx context.Context, key interface{}, f Func, duration time.Duration) (*gvar.Var, error) {
	defer c.handleLruKey(ctx, key)
	v, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if v == nil {
		value, err := f(ctx)
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, nil
		}
		return c.doSetWithLockCheck(ctx, key, value, duration)
	}
	return v, nil
}

// GetOrSetFuncLock 检索并返回 `key` 的值，如果 `key` 不存在于缓存中，则使用函数 `f` 的结果设置 `key` 并返回其结果。
// 键值对在 `duration` 时间后过期。
//
// 如果 `duration` == 0，则永不过期。
// 如果 `duration` < 0 或者给定的 `value` 为 nil，则删除 `key`，但如果 `value` 是函数且函数结果为 nil，则不做任何操作。
//
// 注意：与函数 `GetOrSetFunc` 的不同之处在于，函数 `f` 在写锁内执行，以保证并发安全。
func (c *AdapterMemory) GetOrSetFuncLock(ctx context.Context, key interface{}, f Func, duration time.Duration) (*gvar.Var, error) {
	defer c.handleLruKey(ctx, key)
	v, err := c.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return c.doSetWithLockCheck(ctx, key, f, duration)
	}
	return v, nil
}

// Contains 检查并返回 true 如果 `key` 存在于缓存中，否则返回 false。
func (c *AdapterMemory) Contains(ctx context.Context, key interface{}) (bool, error) {
	v, err := c.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return v != nil, nil
}

// GetExpire 检索并返回缓存中 `key` 的过期时间。
//
// 注意：
// 如果 `key` 永不过期，返回 0。
// 如果 `key` 不存在于缓存中，返回 -1。
func (c *AdapterMemory) GetExpire(ctx context.Context, key interface{}) (time.Duration, error) {
	if item, ok := c.data.Get(key); ok {
		c.handleLruKey(ctx, key)
		return time.Duration(item.e-gtime.TimestampMilli()) * time.Millisecond, nil
	}
	return -1, nil
}

// Remove 从缓存中删除一个或多个键，并返回其值。
// 如果给定多个键，返回最后一个被删除项的值。
func (c *AdapterMemory) Remove(ctx context.Context, keys ...interface{}) (*gvar.Var, error) {
	defer c.lru.Remove(keys...)
	return c.doRemove(ctx, keys...)
}

func (c *AdapterMemory) doRemove(_ context.Context, keys ...interface{}) (*gvar.Var, error) {
	var removedKeys []interface{}
	removedKeys, value, err := c.data.Remove(keys...)
	if err != nil {
		return nil, err
	}
	for _, key := range removedKeys {
		c.eventList.PushBack(&adapterMemoryEvent{
			k: key,
			e: gtime.TimestampMilli() - 1000,
		})
	}
	return gvar.New(value), nil
}

// Update 更新 `key` 的值而不改变其过期时间，并返回旧值。
// 如果 `key` 不存在于缓存中，返回的值 `exist` 为 false。
//
// 如果给定的 `value` 为 nil，则删除 `key`。
// 如果 `key` 不存在于缓存中，则不做任何操作。
func (c *AdapterMemory) Update(ctx context.Context, key interface{}, value interface{}) (oldValue *gvar.Var, exist bool, err error) {
	v, exist, err := c.data.Update(key, value)
	if exist {
		c.handleLruKey(ctx, key)
	}
	return gvar.New(v), exist, err
}

// UpdateExpire 更新 `key` 的过期时间，并返回旧的过期时间值。
//
// 如果 `key` 不存在于缓存中，返回 -1 且不做任何操作。
// 如果 `duration` < 0，则删除 `key`。
func (c *AdapterMemory) UpdateExpire(ctx context.Context, key interface{}, duration time.Duration) (oldDuration time.Duration, err error) {
	newExpireTime := c.getInternalExpire(duration)
	oldDuration, err = c.data.UpdateExpire(key, newExpireTime)
	if err != nil {
		return
	}
	if oldDuration != -1 {
		c.eventList.PushBack(&adapterMemoryEvent{
			k: key,
			e: newExpireTime,
		})
		c.handleLruKey(ctx, key)
	}
	return
}

// Size 返回缓存的大小。
func (c *AdapterMemory) Size(ctx context.Context) (size int, err error) {
	return c.data.Size()
}

// Data 以映射类型返回缓存中所有键值对的副本。
func (c *AdapterMemory) Data(ctx context.Context) (map[interface{}]interface{}, error) {
	return c.data.Data()
}

// Keys 以切片形式返回缓存中的所有键。
func (c *AdapterMemory) Keys(ctx context.Context) ([]interface{}, error) {
	return c.data.Keys()
}

// Values 以切片形式返回缓存中的所有值。
func (c *AdapterMemory) Values(ctx context.Context) ([]interface{}, error) {
	return c.data.Values()
}

// Clear 清除缓存中的所有数据。
// 注意：此函数较敏感，应谨慎使用。
func (c *AdapterMemory) Clear(ctx context.Context) error {
	c.data.Clear()
	c.lru.Clear()
	return nil
}

// Close 关闭缓存。
func (c *AdapterMemory) Close(ctx context.Context) error {
	c.closed.Set(true)
	return nil
}

// doSetWithLockCheck 如果 `key` 不存在于缓存中，则使用 `key`-`value` 对设置缓存，
// 在 `duration` 时间后过期。
//
// 如果 `duration` == 0，则永不过期。
// 参数 `value` 可以是 <func() interface{}> 类型，但如果函数结果为 nil，则不做任何操作。
//
// 在设置到缓存之前，使用互斥写锁双重检查 `key` 是否存在于缓存中。
func (c *AdapterMemory) doSetWithLockCheck(ctx context.Context, key interface{}, value interface{}, duration time.Duration) (result *gvar.Var, err error) {
	expireTimestamp := c.getInternalExpire(duration)
	v, err := c.data.SetWithLock(ctx, key, value, expireTimestamp)
	c.eventList.PushBack(&adapterMemoryEvent{k: key, e: expireTimestamp})
	return gvar.New(v), err
}

// getInternalExpire 将给定的过期持续时间转换为毫秒并返回。
func (c *AdapterMemory) getInternalExpire(duration time.Duration) int64 {
	if duration == 0 {
		return defaultMaxExpire
	}
	return gtime.TimestampMilli() + duration.Nanoseconds()/1000000
}

// makeExpireKey 将 `expire`（毫秒）分组到其对应的秒。
func (c *AdapterMemory) makeExpireKey(expire int64) int64 {
	return int64(math.Ceil(float64(expire/1000)+1) * 1000)
}

// syncEventAndClearExpired 执行异步任务循环：
//  1. 异步处理事件列表中的数据，并将结果同步到 `expireTimes` 和 `expireSets` 属性。
//  2. 清理过期的键值对数据。
func (c *AdapterMemory) syncEventAndClearExpired(ctx context.Context) {
	if c.closed.Val() {
		gtimer.Exit()
		return
	}
	var (
		event         *adapterMemoryEvent
		oldExpireTime int64
		newExpireTime int64
	)
	// ================================
	// 数据过期同步。
	// ================================
	for {
		v := c.eventList.PopFront()
		if v == nil {
			break
		}
		event = v.(*adapterMemoryEvent)
		// 获取旧的过期集合。
		oldExpireTime = c.expireTimes.Get(event.k)
		// 计算新的过期时间集合。
		newExpireTime = c.makeExpireKey(event.e)
		// 此键的过期时间已更改。
		if newExpireTime != oldExpireTime {
			c.expireSets.GetOrNew(newExpireTime).Add(event.k)
			if oldExpireTime != 0 {
				c.expireSets.GetOrNew(oldExpireTime).Remove(event.k)
			}
			// 更新 `event.k` 的过期时间。
			c.expireTimes.Set(event.k, newExpireTime)
		}
	}
	// =================================
	// 数据过期自动清理。
	// =================================
	var (
		expireSet  *gset.Set
		expireTime int64
		currentEk  = c.makeExpireKey(gtime.TimestampMilli())
	)
	// 自动移除最近几秒的过期键集合。
	for i := int64(1); i <= 5; i++ {
		expireTime = currentEk - i*1000
		if expireSet = c.expireSets.Get(expireTime); expireSet != nil {
			// 遍历集合以删除其中的所有键。
			expireSet.Iterator(func(key interface{}) bool {
				c.deleteExpiredKey(key)
				// 为 lru 移除自动过期的键。
				c.lru.Remove(key)
				return true
			})
			// 在删除其所有键后删除该集合。
			c.expireSets.Delete(expireTime)
		}
	}
}

func (c *AdapterMemory) handleLruKey(ctx context.Context, keys ...interface{}) {
	if c.lru == nil {
		return
	}
	if evictedKeys := c.lru.SaveAndEvict(keys...); len(evictedKeys) > 0 {
		_, _ = c.doRemove(ctx, evictedKeys...)
		return
	}
	return
}

// deleteExpiredKey 删除给定 `key` 的键值对。
// 参数 `force` 指定是否强制执行此删除操作。
func (c *AdapterMemory) deleteExpiredKey(key interface{}) {
	// 在真正从缓存中删除之前双重检查。
	c.data.Delete(key)
	// 从 `expireTimes` 中删除其过期时间。
	c.expireTimes.Delete(key)
}
