package gcache

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gset"
	"sync"
)

type memoryExpireSets struct {
	// expireSetMu 确保 expireSets 映射的并发安全性。
	mu sync.RWMutex
	// expireSets 是其密钥集映射的过期时间戳（以秒数计），用于快速索引和删除。
	expireSets map[int64]*gset.Set
}

func newMemoryExpireSets() *memoryExpireSets {
	return &memoryExpireSets{
		expireSets: make(map[int64]*gset.Set),
	}
}

func (d *memoryExpireSets) Get(key int64) (result *gset.Set) {
	d.mu.RLock()
	result = d.expireSets[key]
	d.mu.RUnlock()
	return
}

func (d *memoryExpireSets) GetOrNew(key int64) (result *gset.Set) {
	if result = d.Get(key); result != nil {
		return
	}
	d.mu.Lock()
	if es, ok := d.expireSets[key]; ok {
		result = es
	} else {
		result = gset.New(true)
		d.expireSets[key] = result
	}
	d.mu.Unlock()
	return
}

func (d *memoryExpireSets) Delete(key int64) {
	d.mu.Lock()
	delete(d.expireSets, key)
	d.mu.Unlock()
}
