package datacenter

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	keyGlobalRunning = "datacenter:running:global"
	keyTenantRunning = "datacenter:running:tenant:%d"
	keyUserDaily     = "datacenter:daily:user:%d:%s"
)

// Limiter 导入导出并发与每日限额（Redis）
type Limiter struct {
	rds *redis.Redis
	s   Settings
}

func NewLimiter(rds *redis.Redis, s Settings) *Limiter {
	return &Limiter{rds: rds, s: s}
}

// Acquire 尝试占用并发槽；daily 仅计数不阻塞并发（S2 简化）
func (l *Limiter) Acquire(ctx context.Context, businessId int64, userId uint64) error {
	if l == nil || l.rds == nil {
		return nil
	}
	if err := l.checkDaily(userId); err != nil {
		return err
	}
	if l.s.GlobalMaxRunning > 0 {
		n, err := l.rds.Incr(keyGlobalRunning)
		if err != nil {
			return err
		}
		if int(n) > l.s.GlobalMaxRunning {
			_, _ = l.rds.Decr(keyGlobalRunning)
			return fmt.Errorf("系统导入导出繁忙，请稍后再试（全局并发已满）")
		}
	}
	if l.s.TenantMaxRunning > 0 && businessId > 0 {
		key := fmt.Sprintf(keyTenantRunning, businessId)
		n, err := l.rds.Incr(key)
		if err != nil {
			l.releaseGlobal()
			return err
		}
		if int(n) > l.s.TenantMaxRunning {
			_, _ = l.rds.Decr(key)
			l.releaseGlobal()
			return fmt.Errorf("当前租户导入导出任务过多，请稍后再试")
		}
	}
	return nil
}

// Release 释放并发槽
func (l *Limiter) Release(businessId int64) {
	if l == nil || l.rds == nil {
		return
	}
	l.releaseGlobal()
	if businessId > 0 && l.s.TenantMaxRunning > 0 {
		key := fmt.Sprintf(keyTenantRunning, businessId)
		n, err := l.rds.Decr(key)
		if err == nil && n < 0 {
			_ = l.rds.Set(key, "0")
		}
	}
}

func (l *Limiter) releaseGlobal() {
	if l.s.GlobalMaxRunning <= 0 {
		return
	}
	n, err := l.rds.Decr(keyGlobalRunning)
	if err == nil && n < 0 {
		_ = l.rds.Set(keyGlobalRunning, "0")
	}
}

func (l *Limiter) checkDaily(userId uint64) error {
	if l.s.UserDailyMaxTasks <= 0 || userId == 0 {
		return nil
	}
	day := time.Now().Format("20060102")
	key := fmt.Sprintf(keyUserDaily, userId, day)
	n, err := l.rds.Incr(key)
	if err != nil {
		return err
	}
	if n == 1 {
		l.rds.Expire(key, 86400)
	}
	if int(n) > l.s.UserDailyMaxTasks {
		_, _ = l.rds.Decr(key)
		return fmt.Errorf("已超过今日导入导出次数上限（%d 次）", l.s.UserDailyMaxTasks)
	}
	return nil
}
