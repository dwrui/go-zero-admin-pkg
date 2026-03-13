package gcache

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/glist"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gmap"
	"sync"
)

// memoryLru 持有 LRU 缓存的信息。
// 它使用 stdlib 中的 list.List 作为其底层的双端链表。
type memoryLru struct {
	mu   sync.RWMutex // Mutex 确保并行安全。
	cap  int          // LRU cap.
	data *gmap.Map    // 密钥映射到列表中的项目。
	list *glist.List  // Key list.
}

// newMemoryLru 创建并返回一个新的 LRU 管理器。
func newMemoryLru(cap int) *memoryLru {
	lru := &memoryLru{
		cap:  cap,
		data: gmap.New(false),
		list: glist.New(false),
	}
	return lru
}

// 移除是从“lru”中删除“密钥”。
func (l *memoryLru) Remove(keys ...interface{}) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, key := range keys {
		if v := l.data.Remove(key); v != nil {
			l.list.Remove(v.(*glist.Element))
		}
	}
}

// SaveAndEvict 把密钥存入 LRU，驱逐并归还备用密钥。
func (l *memoryLru) SaveAndEvict(keys ...interface{}) (evictedKeys []interface{}) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	evictedKeys = make([]interface{}, 0)
	for _, key := range keys {
		if evictedKey := l.doSaveAndEvict(key); evictedKey != nil {
			evictedKeys = append(evictedKeys, evictedKey)
		}
	}
	return
}

func (l *memoryLru) doSaveAndEvict(key interface{}) (evictedKey interface{}) {
	var element *glist.Element
	if v := l.data.Get(key); v != nil {
		element = v.(*glist.Element)
		if element.Prev() == nil {
			// It this element is already on top of list,
			// it ignores the element moving.
			return
		}
		l.list.Remove(element)
	}

	// 将激活键推到列表顶部。
	element = l.list.PushFront(key)
	l.data.Set(key, element)
	// 从列表中移除备用钥匙。
	if l.data.Size() <= l.cap {
		return
	}

	if evictedKey = l.list.PopBack(); evictedKey != nil {
		l.data.Remove(evictedKey)
	}
	return
}

// 清除键会删除所有键。
func (l *memoryLru) Clear() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.data.Clear()
	l.list.Clear()
}
