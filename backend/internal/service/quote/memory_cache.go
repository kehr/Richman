package quote

import (
	"sync"
	"time"
)

type cacheEntry struct {
	dto       *QuoteDTO
	expiresAt time.Time
}

type memoryCache struct {
	mu    sync.RWMutex
	store map[string]cacheEntry
}

func newMemoryCache() *memoryCache {
	return &memoryCache{
		store: make(map[string]cacheEntry),
	}
}

// Get returns a cached QuoteDTO if it exists and has not expired.
func (c *memoryCache) Get(key string) (*QuoteDTO, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.store[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.dto, true
}

// Set stores a QuoteDTO with the given TTL.
func (c *memoryCache) Set(key string, dto *QuoteDTO, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = cacheEntry{dto: dto, expiresAt: time.Now().Add(ttl)}
}
