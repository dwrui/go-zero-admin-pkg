package cache

import (
	"time"
)

type Options struct {
	// 基础 TTL
	TTL time.Duration
	// 空值缓存 TTL（防穿透）
	NullTTL time.Duration
	// 是否启用击穿保护（重建锁）
	// EnableRebuildLock 启用分布式缓存重建互斥锁（仅在 10+ 实例 / 热点 key 可能击穿 DB 时开启）
	// 开了以后缓存 miss 时多一次 SetnxEx RTT，失败降级到自旋等待，整体 RT 略有上升
	// 小集群（≤ 5 实例）/ DB 连接池富余 → 建议 false，仅靠 SingleFlight 足够
	EnableRebuildLock bool
	// 重建锁 TTL
	RebuildLockTTL time.Duration
	// 重建锁重试间隔
	RebuildLockRetryInterval time.Duration
	// 重建锁最大重试次数
	RebuildLockMaxRetries int
	// 是否启用本地缓存
	EnableLocalCache bool
	// 本地缓存最大条目数
	LocalCacheMaxEntries int
	// 本地缓存 TTL
	LocalCacheTTL time.Duration
}

type Option func(*Options)

func WithTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.TTL = ttl
	}
}

func WithNullTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.NullTTL = ttl
	}
}

func WithRebuildLock(enable bool) Option {
	return func(o *Options) {
		o.EnableRebuildLock = enable
	}
}

func WithRebuildLockTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.RebuildLockTTL = ttl
	}
}

func WithRebuildLockRetry(interval time.Duration, maxRetries int) Option {
	return func(o *Options) {
		o.RebuildLockRetryInterval = interval
		o.RebuildLockMaxRetries = maxRetries
	}
}

func WithLocalCache(enable bool, maxEntries int, ttl time.Duration) Option {
	return func(o *Options) {
		o.EnableLocalCache = enable
		o.LocalCacheMaxEntries = maxEntries
		o.LocalCacheTTL = ttl
	}
}

func defaultOptions() *Options {
	return &Options{
		TTL:                      5 * time.Minute,
		NullTTL:                  1 * time.Minute,
		EnableRebuildLock:        false,
		RebuildLockTTL:           5 * time.Second,
		RebuildLockRetryInterval: 50 * time.Millisecond,
		RebuildLockMaxRetries:    10,
		EnableLocalCache:         false,
		LocalCacheMaxEntries:     10000,
		LocalCacheTTL:            30 * time.Second,
	}
}
