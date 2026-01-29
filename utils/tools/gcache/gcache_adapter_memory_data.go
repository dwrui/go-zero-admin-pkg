package gcache

import (
	"context"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gtime"
	"sync"
	"time"
)

type memoryData struct {
	mu   sync.RWMutex                   // dataMu 确保底层数据映射的并发安全性。
	data map[interface{}]memoryDataItem // data 是存储在哈希表中的底层缓存数据。
}

// memoryDataItem 保存内部缓存项数据。
type memoryDataItem struct {
	v interface{} // Value.
	e int64       // Expire timestamp in milliseconds.
}

func newMemoryData() *memoryData {
	return &memoryData{
		data: make(map[interface{}]memoryDataItem),
	}
}

// Update 更新 `key` 的值而不改变其过期时间，并返回旧值。
// 如果 `key` 不存在于缓存中，则返回的 `exist` 为 false。
//
// 如果给定 `value` 为 nil，则删除 `key`。
// 如果 `key` 不存在于缓存中，则不执行任何操作。
func (d *memoryData) Update(key interface{}, value interface{}) (oldValue interface{}, exist bool, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if item, ok := d.data[key]; ok {
		d.data[key] = memoryDataItem{
			v: value,
			e: item.e,
		}
		return item.v, true, nil
	}
	return nil, false, nil
}

// UpdateExpire 更新“密钥”的过期时间，并返回旧的过期时长值.
//
// 如果 `key` 不存在于缓存中，则返回 -1 并不执行任何操作。
// 如果 `duration` < 0，则删除 `key`。
func (d *memoryData) UpdateExpire(key interface{}, expireTime int64) (oldDuration time.Duration, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if item, ok := d.data[key]; ok {
		d.data[key] = memoryDataItem{
			v: item.v,
			e: expireTime,
		}
		return time.Duration(item.e-gtime.TimestampMilli()) * time.Millisecond, nil
	}
	return -1, nil
}

// 移除会删除缓存中的一个或多个键，并返回其值。
// 如果给出多个键，则返回被删除的最后一项的值。
func (d *memoryData) Remove(keys ...interface{}) (removedKeys []interface{}, value interface{}, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	removedKeys = make([]interface{}, 0)
	for _, key := range keys {
		item, ok := d.data[key]
		if ok {
			value = item.v
			delete(d.data, key)
			removedKeys = append(removedKeys, key)
		}
	}
	return removedKeys, value, nil
}

// Data 返回缓存中所有键值对的副本，作为 map 类型。
func (d *memoryData) Data() (map[interface{}]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var (
		data     = make(map[interface{}]interface{}, len(d.data))
		nowMilli = gtime.TimestampMilli()
	)
	for k, v := range d.data {
		if v.e > nowMilli {
			data[k] = v.v
		}
	}
	return data, nil
}

// Keys 返回缓存中所有键的副本，作为 slice 类型。
func (d *memoryData) Keys() ([]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var (
		keys     = make([]interface{}, 0, len(d.data))
		nowMilli = gtime.TimestampMilli()
	)
	for k, v := range d.data {
		if v.e > nowMilli {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

// Values 返回缓存中所有值的副本，作为 slice 类型。
func (d *memoryData) Values() ([]interface{}, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var (
		values   = make([]interface{}, 0, len(d.data))
		nowMilli = gtime.TimestampMilli()
	)
	for _, v := range d.data {
		if v.e > nowMilli {
			values = append(values, v.v)
		}
	}
	return values, nil
}

// Size 返回缓存中未过期项的数量。
func (d *memoryData) Size() (size int, err error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var nowMilli = gtime.TimestampMilli()
	for _, v := range d.data {
		if v.e > nowMilli {
			size++
		}
	}
	return size, nil
}

// Clear 清除缓存中的所有数据项。
// 注意：此函数敏感，应谨慎使用。
func (d *memoryData) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.data = make(map[interface{}]memoryDataItem)
}

func (d *memoryData) Get(key interface{}) (item memoryDataItem, ok bool) {
	d.mu.RLock()
	item, ok = d.data[key]
	d.mu.RUnlock()
	return
}

func (d *memoryData) Set(key interface{}, value memoryDataItem) {
	d.mu.Lock()
	d.data[key] = value
	d.mu.Unlock()
}

// SetMap 批量设置缓存中的多个键值对，每个键值对在 `duration` 后过期。
//
// 如果 `duration` == 0，则键值对不会过期。
// 如果 `duration` < 0 或给定 `value` 为 nil，则删除 `data` 中的键。
func (d *memoryData) SetMap(data map[interface{}]interface{}, expireTime int64) error {
	d.mu.Lock()
	for k, v := range data {
		d.data[k] = memoryDataItem{
			v: v,
			e: expireTime,
		}
	}
	d.mu.Unlock()
	return nil
}

func (d *memoryData) SetWithLock(ctx context.Context, key interface{}, value interface{}, expireTimestamp int64) (interface{}, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var (
		err error
	)
	if v, ok := d.data[key]; ok && !v.IsExpired() {
		return v.v, nil
	}
	f, ok := value.(Func)
	if !ok {
		// 与原始函数值兼容。
		f, ok = value.(func(ctx context.Context) (value interface{}, err error))
	}
	if ok {
		if value, err = f(ctx); err != nil {
			return nil, err
		}
		if value == nil {
			return nil, nil
		}
	}
	d.data[key] = memoryDataItem{v: value, e: expireTimestamp}
	return value, nil
}

func (d *memoryData) Delete(key interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.data, key)
}
