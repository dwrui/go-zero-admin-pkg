package gcache

import (
	"context"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
)

// Cache struct.
type Cache struct {
	localAdapter
}

// localAdapter 是 Adapter 的别名，仅用于嵌入属性。
type localAdapter = Adapter

// 新建时使用默认的内存适配器创建并返回新的缓存对象。
// 注意，LRU功能仅通过内存适配器实现。
func New(lruCap ...int) *Cache {
	var adapter Adapter
	if len(lruCap) == 0 {
		adapter = NewAdapterMemory()
	} else {
		adapter = NewAdapterMemoryLru(lruCap[0])
	}
	c := &Cache{
		localAdapter: adapter,
	}
	return c
}

// NewWithAdapter 创建并返回一个 Cache 对象，该对象使用给定的 Adapter 实现。
func NewWithAdapter(adapter Adapter) *Cache {
	return &Cache{
		localAdapter: adapter,
	}
}

// SetAdapter 为当前缓存设置适配器。
// 请注意，此设置函数不是并发安全的，这意味着您不应该在多个 goroutine 中并发调用此设置函数。
func (c *Cache) SetAdapter(adapter Adapter) {
	c.localAdapter = adapter
}

// GetAdapter 返回当前缓存中设置的适配器。
func (c *Cache) GetAdapter() Adapter {
	return c.localAdapter
}

// Removes 删除缓存中的 `keys`。
func (c *Cache) Removes(ctx context.Context, keys []interface{}) error {
	_, err := c.Remove(ctx, keys...)
	return err
}

// KeyStrings 返回缓存中的所有键作为字符串切片。
func (c *Cache) KeyStrings(ctx context.Context) ([]string, error) {
	keys, err := c.Keys(ctx)
	if err != nil {
		return nil, err
	}
	return gconv.Strings(keys), nil
}
