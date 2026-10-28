package cache

import (
	"container/list"
	"sync"
	"time"
)

// MediaItem represents a cached media file in memory.
type MediaItem struct {
	ID        string
	Filename  string
	MimeType  string
	Data      []byte
	CreatedAt time.Time
	Size      int64
}

type mediaElement struct {
	key  string
	item MediaItem
}

// MediaCache is a thread-safe in-memory LRU cache with byte-level memory bounds.
type MediaCache struct {
	mu          sync.RWMutex
	items       map[string]*list.Element
	evictList   *list.List
	maxBytes    int64
	currentSize int64
}

// NewMediaCache creates a new MediaCache with the specified max capacity in bytes.
func NewMediaCache(maxBytes int64) *MediaCache {
	if maxBytes <= 0 {
		maxBytes = 256 * 1024 * 1024 // 256 MB default
	}
	return &MediaCache{
		items:     make(map[string]*list.Element),
		evictList: list.New(),
		maxBytes:  maxBytes,
	}
}

// Get retrieves a MediaItem by ID, promoting it to the most recently used position.
func (c *MediaCache) Get(id string) (MediaItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[id]; ok {
		c.evictList.MoveToFront(elem)
		return elem.Value.(*mediaElement).item, true
	}
	return MediaItem{}, false
}

// Set stores a MediaItem by ID, evicting the least recently used items if capacity is exceeded.
func (c *MediaCache) Set(id string, item MediaItem) {
	// If a single item exceeds 35% of total cache capacity, do not cache to prevent thrashing.
	if item.Size > c.maxBytes*35/100 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[id]; ok {
		// Replace existing item
		oldElem := elem.Value.(*mediaElement)
		c.currentSize -= oldElem.item.Size
		oldElem.item = item
		c.currentSize += item.Size
		c.evictList.MoveToFront(elem)
	} else {
		// Insert new item at the front
		elem := c.evictList.PushFront(&mediaElement{key: id, item: item})
		c.items[id] = elem
		c.currentSize += item.Size
	}

	// Evict oldest items if current memory exceeds maxBytes
	for c.currentSize > c.maxBytes && c.evictList.Len() > 0 {
		oldest := c.evictList.Back()
		if oldest != nil {
			c.removeElement(oldest)
		}
	}
}

// Delete removes an item from the cache.
func (c *MediaCache) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[id]; ok {
		c.removeElement(elem)
	}
}

func (c *MediaCache) removeElement(elem *list.Element) {
	c.evictList.Remove(elem)
	entry := elem.Value.(*mediaElement)
	delete(c.items, entry.key)
	c.currentSize -= entry.item.Size
}

// CurrentSize returns the total number of cached bytes in memory.
func (c *MediaCache) CurrentSize() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentSize
}

// Len returns the count of items in the cache.
func (c *MediaCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear evicts every cached item after bulk media deletion.
func (c *MediaCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element)
	c.evictList.Init()
	c.currentSize = 0
}
