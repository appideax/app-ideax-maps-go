package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
	Delete(key string)
}

type memoryCache struct {
	c *cache.Cache
}

func NewMemoryCache(defaultTTL time.Duration) Cache {
	cleanupInterval := defaultTTL
	if cleanupInterval < time.Minute {
		cleanupInterval = time.Minute
	}
	return &memoryCache{c: cache.New(defaultTTL, cleanupInterval)}
}

func (m *memoryCache) Get(key string) ([]byte, bool) {
	value, found := m.c.Get(key)
	if !found {
		return nil, false
	}
	b, ok := value.([]byte)
	return b, ok
}

func (m *memoryCache) Set(key string, value []byte, ttl time.Duration) {
	m.c.Set(key, value, ttl)
}

func (m *memoryCache) Delete(key string) {
	m.c.Delete(key)
}
