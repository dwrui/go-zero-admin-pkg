package cache

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

var (
	ErrLockNotAcquired = errors.New("lock: not acquired")
)

// RedisLock 分布式锁对象
type RedisLock struct {
	rds   *redis.Redis
	key   string
	value string // 锁持有者标识（UUID），用于安全释放
	ttl   time.Duration
}

// NewRedisLock 创建锁实例
// key 建议传入带 {hash_tag} 的 key，例如 "lock:order:{123}"
func NewRedisLock(rds *redis.Redis, key string, ttl time.Duration) *RedisLock {
	return &RedisLock{
		rds:   rds,
		key:   key,
		value: newUUID(), // 每次加锁生成唯一标识
		ttl:   ttl,
	}
}

// TryLock 尝试加锁
func (l *RedisLock) TryLock(ctx context.Context) error {
	// 使用 SetnxEx 原子操作：不存在则设置，并设置过期时间
	ok, err := l.rds.SetnxExCtx(ctx, l.key, l.value, int(l.ttl.Seconds()))
	if err != nil {
		logx.WithContext(ctx).Errorf("redis lock error: %v", err)
		return err
	}
	if !ok {
		return ErrLockNotAcquired
	}
	return nil
}

// Unlock 安全释放锁（Lua 脚本保证原子性：只能删除自己加的锁）
func (l *RedisLock) Unlock(ctx context.Context) error {
	// Lua 脚本：仅当 value 匹配时才删除
	_, err := l.rds.EvalCtx(ctx, unlockScript, []string{l.key}, l.value)
	if err != nil {
		logx.WithContext(ctx).Errorf("redis unlock error: %v", err)
	}
	return err
}

// LockWait 等待式加锁（自旋尝试，用于业务层调用）
func (l *RedisLock) LockWait(ctx context.Context, retryInterval time.Duration, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := l.TryLock(ctx); err == nil {
			return nil
		} else if errors.Is(err, ErrLockNotAcquired) {
			// 未拿到锁，等待后重试
			time.Sleep(retryInterval)
			continue
		} else {
			// Redis 异常
			return err
		}
	}
	return ErrLockNotAcquired
}
