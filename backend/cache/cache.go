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

type Cache struct {
	mu         sync.RWMutex
	entries    map[string]entry
	ttl        time.Duration
	maxEntries int
	stop       chan struct{}
	done       chan struct{}
	closeOnce  sync.Once
}

func New(ttl time.Duration) *Cache {
	c := &Cache{
		entries:    make(map[string]entry),
		ttl:        ttl,
		maxEntries: defaultMaxEntries,
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
	go c.janitor()
	return c
}

func (c *Cache) Close() {
	if c.stop == nil {
		return
	}
	c.closeOnce.Do(func() { close(c.stop) })
	<-c.done
}

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

func (c *Cache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxEntries {
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
	defer close(c.done)
	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
		}
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
