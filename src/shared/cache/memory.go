package cache

import (
	"container/list"
	"sync"
	"time"
)

type Cache interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
	Delete(key string)
}

type entry struct {
	key       string
	value     []byte
	expiresAt time.Time
}

// memoryCache es un cache LRU con tope de memoria: al pasar maxBytes descarta
// lo que lleva mas tiempo sin usarse, para que las teselas no crezcan sin fin.
type memoryCache struct {
	mu       sync.Mutex
	maxBytes int64
	bytes    int64
	order    *list.List
	items    map[string]*list.Element
}

func NewMemoryCache(maxBytes int64) Cache {
	return &memoryCache{
		maxBytes: maxBytes,
		order:    list.New(),
		items:    make(map[string]*list.Element),
	}
}

func (m *memoryCache) Get(key string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	element, found := m.items[key]
	if !found {
		return nil, false
	}
	item := element.Value.(*entry)
	if time.Now().After(item.expiresAt) {
		m.remove(element)
		return nil, false
	}
	m.order.MoveToFront(element)
	return item.value, true
}

func (m *memoryCache) Set(key string, value []byte, ttl time.Duration) {
	size := int64(len(value))
	if ttl <= 0 || size > m.maxBytes {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if element, found := m.items[key]; found {
		m.remove(element)
	}
	m.items[key] = m.order.PushFront(&entry{key: key, value: value, expiresAt: time.Now().Add(ttl)})
	m.bytes += size

	for m.bytes > m.maxBytes {
		m.remove(m.order.Back())
	}
}

func (m *memoryCache) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if element, found := m.items[key]; found {
		m.remove(element)
	}
}

func (m *memoryCache) remove(element *list.Element) {
	item := m.order.Remove(element).(*entry)
	delete(m.items, item.key)
	m.bytes -= int64(len(item.value))
}
