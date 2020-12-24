package memcache

import (
	"sync"

	"github.com/fusidic/FuCache/pkg/lru"
)

type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 延迟创建，当第一次调用add方法的时候再创建LRU
	if c.lru == nil {
		c.lru = lru.NewLRU(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	// 加上了并发读写的锁
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	// 全都是封装
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}
