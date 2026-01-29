package gcache

import (
	"sync"
)

type memoryExpireTimes struct {
	mu          sync.RWMutex          // expireTimeMu 确保 expireTimes 映射的并发安全性。
	expireTimes map[interface{}]int64 // expireTimes 是其过期时间戳（以毫秒数计）的映射，用于快速索引和删除。
}

func newMemoryExpireTimes() *memoryExpireTimes {
	return &memoryExpireTimes{
		expireTimes: make(map[interface{}]int64),
	}
}

func (d *memoryExpireTimes) Get(key interface{}) (value int64) {
	d.mu.RLock()
	value = d.expireTimes[key]
	d.mu.RUnlock()
	return
}

func (d *memoryExpireTimes) Set(key interface{}, value int64) {
	d.mu.Lock()
	d.expireTimes[key] = value
	d.mu.Unlock()
}

func (d *memoryExpireTimes) Delete(key interface{}) {
	d.mu.Lock()
	delete(d.expireTimes, key)
	d.mu.Unlock()
}
