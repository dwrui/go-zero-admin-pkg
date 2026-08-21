package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/syncx"
)

var (
	ErrNotFound = errors.New("cache: not found")
	ErrLoadFunc = errors.New("cache: load function error")
)

// Loader 数据加载函数（查 DB / RPC）
type Loader func(ctx context.Context) (any, error)

// Cache 统一缓存接口
type Cache struct {
	rds        *redis.Redis
	opts       *Options
	localCache *collection.Cache // go-zero 内置 LRU：线程安全 + 自动过期清理 + 容量淘汰
	sfGroup    syncx.SingleFlight
	ctx        context.Context
	cancel     context.CancelFunc
	rngMu      sync.Mutex // 保护 rand.Source
	rng        *rand.Rand
}

// NewCache 创建 Cache 实例
func NewCache(rds *redis.Redis, opts ...Option) *Cache {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Cache{
		rds:     rds,
		opts:    o,
		sfGroup: syncx.NewSingleFlight(),
		ctx:     ctx,
		cancel:  cancel,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if o.EnableLocalCache {
		// collection.NewCache(expire) 创建本地缓存
		//   - expire：默认 TTL（Set 时生效，SetWithExpire 可覆盖）
		//   - WithLimit：容量上限，超过按 LRU 淘汰
		//   - 内置后台协程自动清理过期 key，无需手写 ticker
		lc, err := collection.NewCache(
			o.LocalCacheTTL,
			collection.WithLimit(o.LocalCacheMaxEntries),
		)
		if err != nil {
			logx.Errorf("collection.NewCache error: %v", err)
			// 初始化失败则降级：不启用本地缓存，不影响 Redis 主流程
			o.EnableLocalCache = false
			c.opts = o
		} else {
			c.localCache = lc
		}
	}

	return c
}

// Stop 释放资源，取消后台上下文
func (c *Cache) Stop() {
	c.cancel()
}

// Get 核心方法：查缓存 → 未命中 → 加载 → 回填
// key: Redis key（不带 env 前缀，由调用方拼）
// dest: 目标结构体指针
// loader: 数据加载函数
func (c *Cache) Get(ctx context.Context, key string, dest any, loader Loader) error {
	// 1. 本地缓存（可选）— collection.Cache 已内部实现过期检查 + 线程安全
	if c.opts.EnableLocalCache && c.localCache != nil {
		if v, ok := c.localCache.Get(key); ok {
			if val, ok := v.([]byte); ok {
				if err := json.Unmarshal(val, dest); err == nil {
					return nil
				} else {
					// 本地缓存反序列化失败（struct 升级/脏数据），主动删除避免反复失败
					logx.Errorf("local cache unmarshal error, key=%s, err=%v", key, err)
					c.localCache.Del(key)
				}
			}
		}
	}

	// 2. Redis 缓存
	val, err := c.rds.GetCtx(ctx, key)
	if err == nil {
		// 空值缓存
		if val == nullValue {
			return ErrNotFound
		}

		// 反序列化
		if err := json.Unmarshal([]byte(val), dest); err != nil {
			logx.Errorf("cache unmarshal error, key=%s, err=%v", key, err)
			// 缓存值与当前 struct 不兼容（代码升级/脏数据），主动删除：
			//   避免后续每个请求都重复走 Get + Unmarshal 失败的无用流程
			//   下一次请求直接 Redis miss → 走 SingleFlight 回源 → 写入新格式缓存
			if _, delErr := c.rds.DelCtx(ctx, key); delErr != nil {
				logx.Errorf("cache unmarshal del error, key=%s, err=%v", key, delErr)
			}
			c.delLocal(key)
		} else {
			// 写入本地缓存 — collection.Cache 自动按 LocalCacheTTL 过期
			c.setLocal(key, []byte(val))
			return nil
		}
	}

	if err != nil && err != redis.Nil {
		logx.Errorf("cache get error, key=%s, err=%v", key, err)
	}

	// 3. 缓存未命中 -> 进入 SingleFlight (防止单机并发击穿)
	result, err := c.sfGroup.Do(key, func() (any, error) {
		// Double Check: 进入 SingleFlight 后，再次检查 Redis
		val, err := c.rds.GetCtx(ctx, key)
		if err == nil && val != "" && val != nullValue {
			return val, nil
		}

		// 4. 分布式锁保护 (防止跨进程击穿)
		if c.opts.EnableRebuildLock {
			return c.loadWithLock(ctx, key, loader)
		}

		// 无锁加载
		return c.loadAndSet(ctx, key, loader)
	})

	if err != nil {
		return err
	}

	// 5. 结果处理
	if strVal, ok := result.(string); ok {
		if strVal == nullValue {
			return ErrNotFound
		}
		if err := json.Unmarshal([]byte(strVal), dest); err != nil {
			return err
		}
		c.setLocal(key, []byte(strVal))
		return nil
	}

	return nil
}

// loadWithLock 带分布式锁的数据加载
func (c *Cache) loadWithLock(ctx context.Context, key string, loader Loader) (interface{}, error) {
	// 锁 Key 使用 Hash Tag，确保在 Redis Cluster 下 Key 和 Lock 在同一分片
	lockKey := fmt.Sprintf("lock:rebuild:{%s}", key)
	uuid := c.newUUID()

	// 尝试加锁
	acquired, err := c.rds.SetnxExCtx(ctx, lockKey, uuid, int(c.opts.RebuildLockTTL.Seconds()))
	if err != nil {
		logx.Errorf("acquire rebuild lock error, key=%s, err=%v", lockKey, err)
		// 锁异常降级：直接加载
		return c.loadAndSet(ctx, key, loader)
	}

	// 没拿到锁 -> 降级策略
	if !acquired {
		logx.Infof("lock contention, fallback to load directly, key=%s", key)
		return c.waitAndRetry(ctx, key, loader)
	}

	// 拿到锁 -> defer 解锁
	defer c.releaseLock(ctx, lockKey, uuid)

	// Double Check：拿到锁后再查一次缓存
	val, err := c.rds.GetCtx(ctx, key)
	if err == nil && val != "" && val != nullValue {
		return val, nil
	}

	// 真正加载数据
	return c.loadAndSet(ctx, key, loader)
}

// loadAndSet 纯粹的数据加载与回填逻辑
func (c *Cache) loadAndSet(ctx context.Context, key string, loader Loader) (interface{}, error) {
	data, err := loader(ctx)
	if err != nil {
		logx.Errorf("loader db query fail key=%s err=%v", key, err)
		return "", fmt.Errorf("%w: %v", ErrLoadFunc, err)
	}

	var resultStr string

	// 空值处理 (防穿透)
	if data == nil {
		c.rds.SetexCtx(ctx, key, nullValue, int(c.opts.NullTTL.Seconds()))
		resultStr = nullValue
	} else {
		bytes, err := json.Marshal(data)
		if err != nil {
			return "", err
		}

		// 写 Redis (带 jitter 防雪崩)
		ttl := jitter(c.opts.TTL)
		if err := c.rds.SetexCtx(ctx, key, string(bytes), int(ttl.Seconds())); err != nil {
			logx.Errorf("cache setex error, key=%s, err=%v", key, err)
		}
		resultStr = string(bytes)
	}

	// 本地缓存由外层 Get 方法统一写入，避免重复写
	return resultStr, nil
}

// waitAndRetry 自旋等待其他节点重建缓存
func (c *Cache) waitAndRetry(ctx context.Context, key string, loader Loader) (interface{}, error) {
	retryInterval := c.opts.RebuildLockRetryInterval
	maxRetries := c.opts.RebuildLockMaxRetries
	if retryInterval <= 0 {
		retryInterval = 50 * time.Millisecond
	}
	if maxRetries <= 0 {
		maxRetries = 40 // 兜底：40 * 50ms = 2s
	}

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(retryInterval):
			// 检查缓存是否已经重建
			val, err := c.rds.GetCtx(ctx, key)
			if err == nil && val != "" && val != nullValue {
				return val, nil
			}
			if err != nil && err != redis.Nil {
				// Redis 异常，跳出等待
				break
			}
		}
	}

	// 等待超时，降级策略：直接查询数据库
	logx.Infof("rebuild lock wait timeout, fallback to load directly, key=%s", key)
	return c.loadAndSet(ctx, key, loader)
}

// releaseLock 释放重建锁（Lua）
func (c *Cache) releaseLock(ctx context.Context, lockKey, uuid string) {
	if _, err := c.rds.EvalCtx(ctx, unlockScript, []string{lockKey}, uuid); err != nil {
		logx.Errorf("release rebuild lock error, key=%s, err=%v", lockKey, err)
	}
}

// Delete 删除缓存（用于 Cache Aside 写操作）
func (c *Cache) Delete(ctx context.Context, key string) error {
	c.delLocal(key)
	_, err := c.rds.DelCtx(ctx, key)
	return err
}

// DelayDoubleDelete 延迟双删（用于写操作后）
func (c *Cache) DelayDoubleDelete(ctx context.Context, key string, delay time.Duration) error {
	if err := c.Delete(ctx, key); err != nil {
		return err
	}

	// 使用缓存实例长期存活的 c.ctx 执行二删，避免请求 ctx 被取消（如客户端断开、超时、网关丢弃）
	// 导致延迟删除没执行，出现脏缓存问题
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-c.ctx.Done():
			return
		case <-timer.C:
			_, err := c.rds.DelCtx(c.ctx, key)
			if err != nil {
				logx.Errorf("delay double delete failed key=%s err=%v", key, err)
			}
			if c.opts.EnableLocalCache {
				c.delLocal(key)
			}
		}
	}()

	return nil
}

// ===== 本地缓存相关（封装 collection.Cache，保持方法签名不变）=====

func (c *Cache) setLocal(key string, value []byte) {
	if !c.opts.EnableLocalCache || c.localCache == nil {
		return
	}
	// collection.Cache.Set 使用 NewCache 时传入的 expire（LocalCacheTTL）
	c.localCache.Set(key, value)
}

func (c *Cache) delLocal(key string) {
	if !c.opts.EnableLocalCache || c.localCache == nil {
		return
	}
	c.localCache.Del(key)
}

// ===== 工具函数 =====

// 包级全局随机源（仅供 newUUID 使用）
var (
	globalRngMu sync.Mutex
	globalRng   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func newUUID() string {
	globalRngMu.Lock()
	r := globalRng.Intn(1000000)
	globalRngMu.Unlock()
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), r)
}

func (c *Cache) newUUID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), c.fastRand())
}

func (c *Cache) fastRand() int {
	c.rngMu.Lock()
	defer c.rngMu.Unlock()
	return c.rng.Intn(1000000)
}
