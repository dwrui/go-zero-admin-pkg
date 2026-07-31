package cache

import (
	"math/rand"
	"time"
)

// jitter 给 TTL 加 ±20% 的随机偏移，防止大量 key 同时过期
func jitter(base time.Duration) time.Duration {
	if base <= 0 {
		return base
	}
	// ±20%
	offset := int64(base / 5)
	jitter := rand.Int63n(offset*2) - offset
	result := base + time.Duration(jitter)
	if result < 0 {
		result = base
	}
	return result
}
