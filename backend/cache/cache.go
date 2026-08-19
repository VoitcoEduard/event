package cache

import (
	"strings"
	"sync"
	"time"
)

const defaultMaxEntries = 2048

type entry struct {
	value     []byte
	expiresAt time.Time
}

// Cache is a small thread-safe TTL cache for byte payloads.
// It is meant for hot read paths (e.g. polling) where the same
// response is served many times per second.
type Cache struct {
	mu         sync.RWMutex
	entries    map[string]entry
	ttl        time.Duration
	maxEntries int
}

func New(ttl time.Duration) *Cache {
	c := &Cache{
		entries:    make(map[string]entry),
		ttl:        ttl,
		maxEntries: defaultMaxEntries,
	}
	go c.janitor()
	return c
}

// Get returns the cached value. Expired entries are treated as misses.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expiresAt) {
		c.Delete(key)
		return nil, false
	}
	return e.value, true
}

// Set stores value under key. The caller must not mutate value afterwards.
func (c *Cache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxEntries {
		// Evict an arbitrary entry to bound memory usage.
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}
	c.entries[key] = entry{value: value, expiresAt: time.Now().Add(c.ttl)}
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// DeletePrefix removes all entries whose key starts with prefix.
func (c *Cache) DeletePrefix(prefix string) {
	c.mu.Lock()
	for k := range c.entries {
		if strings.HasPrefix(k, prefix) {
			delete(c.entries, k)
		}
	}
	c.mu.Unlock()
}

func (c *Cache) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}
