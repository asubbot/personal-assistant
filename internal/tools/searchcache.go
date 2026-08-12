package tools

import (
	"container/list"
	"sync"
	"time"
)

// searchCache is an in-memory LRU + TTL cache for web_search (EP-011, REQ-11.009–REQ-11.012).
type searchCache struct {
	mu      sync.Mutex
	max     int
	ttl     time.Duration
	entries map[string]*searchCacheEntry
	lru     *list.List
	now     func() time.Time
}

type searchCacheEntry struct {
	key      string
	payload  string
	storedAt time.Time
	elem     *list.Element
}

func newSearchCache(maxEntries int, ttl time.Duration, now func() time.Time) *searchCache {
	if now == nil {
		now = time.Now
	}
	return &searchCache{
		max:     maxEntries,
		ttl:     ttl,
		entries: make(map[string]*searchCacheEntry),
		lru:     list.New(),
		now:     now,
	}
}

func (c *searchCache) get(key string) (payload string, hit bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		return "", false
	}
	if c.now().Sub(e.storedAt) >= c.ttl {
		c.removeEntry(e)
		return "", false
	}
	c.lru.MoveToFront(e.elem)
	return e.payload, true
}

func (c *searchCache) set(key, payload string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[key]; ok {
		e.payload = payload
		e.storedAt = c.now()
		c.lru.MoveToFront(e.elem)
		return
	}
	for len(c.entries) >= c.max && c.lru.Len() > 0 {
		oldest := c.lru.Back()
		if oldest == nil {
			break
		}
		// type-assertion: safe — this private LRU stores only *searchCacheEntry values
		oe := oldest.Value.(*searchCacheEntry)
		c.removeEntry(oe)
	}
	e := &searchCacheEntry{
		key:      key,
		payload:  payload,
		storedAt: c.now(),
	}
	e.elem = c.lru.PushFront(e)
	c.entries[key] = e
}

func (c *searchCache) removeEntry(e *searchCacheEntry) {
	if e.elem != nil {
		c.lru.Remove(e.elem)
	}
	delete(c.entries, e.key)
}
