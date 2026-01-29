package gcache

import (
	"context"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gvar"
	"time"
)

// MustGet 表现得像 Get，但一旦出现错误就会慌乱。
func (c *Cache) MustGet(ctx context.Context, key interface{}) *gvar.Var {
	v, err := c.Get(ctx, key)
	if err != nil {
		panic(err)
	}
	return v
}

// MustGetOrSet 类似于 GetOrSet，但如果出现错误它会慌乱。
func (c *Cache) MustGetOrSet(ctx context.Context, key interface{}, value interface{}, duration time.Duration) *gvar.Var {
	v, err := c.GetOrSet(ctx, key, value, duration)
	if err != nil {
		panic(err)
	}
	return v
}

// MustGetOrSetFunc 类似于 GetOrSetFunc，但如果出现错误它会慌乱。
func (c *Cache) MustGetOrSetFunc(ctx context.Context, key interface{}, f Func, duration time.Duration) *gvar.Var {
	v, err := c.GetOrSetFunc(ctx, key, f, duration)
	if err != nil {
		panic(err)
	}
	return v
}

// MustGetOrSetFuncLock 的行为类似于 GetOrSetFuncLock，但如果出现错误它会慌乱。
func (c *Cache) MustGetOrSetFuncLock(ctx context.Context, key interface{}, f Func, duration time.Duration) *gvar.Var {
	v, err := c.GetOrSetFuncLock(ctx, key, f, duration)
	if err != nil {
		panic(err)
	}
	return v
}

// MustContains 的表现类似于 Contains，但如果出现错误它会慌乱。
func (c *Cache) MustContains(ctx context.Context, key interface{}) bool {
	v, err := c.Contains(ctx, key)
	if err != nil {
		panic(err)
	}
	return v
}

// MustGetExpire 的表现类似于 GetExpire，但如果出现错误它会慌乱。
func (c *Cache) MustGetExpire(ctx context.Context, key interface{}) time.Duration {
	v, err := c.GetExpire(ctx, key)
	if err != nil {
		panic(err)
	}
	return v
}

// MustSize 的表现类似于 Size，但如果出现错误它会慌乱。
func (c *Cache) MustSize(ctx context.Context) int {
	v, err := c.Size(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// MustData 的表现类似于 Data，但如果出现错误它会慌乱。
func (c *Cache) MustData(ctx context.Context) map[interface{}]interface{} {
	v, err := c.Data(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// MustKeys 的表现类似于 Keys，但如果出现错误它会慌乱。
func (c *Cache) MustKeys(ctx context.Context) []interface{} {
	v, err := c.Keys(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// MustKeyStrings 的表现类似于 KeyStrings，但如果出现错误它会慌乱。
func (c *Cache) MustKeyStrings(ctx context.Context) []string {
	v, err := c.KeyStrings(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// MustValues 的表现类似于 Values，但如果出现错误它会慌乱。
func (c *Cache) MustValues(ctx context.Context) []interface{} {
	v, err := c.Values(ctx)
	if err != nil {
		panic(err)
	}
	return v
}
