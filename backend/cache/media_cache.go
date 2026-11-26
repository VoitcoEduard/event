package cache

import (
	"container/list"
	"sync"
	"time"
)

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

type MediaCache struct {
	mu          sync.RWMutex
	items       map[string]*list.Element
	evictList   *list.List
	maxBytes    int64
	currentSize int64
}

func NewMediaCache(maxBytes int64) *MediaCache {
	if maxBytes <= 0 {
		maxBytes = 256 * 1024 * 1024
	}
	return &MediaCache{
		items:     make(map[string]*list.Element),
		evictList: list.New(),
		maxBytes:  maxBytes,
	}
}

func (c *MediaCache) Get(id string) (MediaItem, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[id]; ok {
		c.evictList.MoveToFront(elem)
		return elem.Value.(*mediaElement).item, true
	}
	return MediaItem{}, false
}

func (c *MediaCache) Set(id string, item MediaItem) {
	if item.Size > c.maxBytes*35/100 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[id]; ok {
		oldElem := elem.Value.(*mediaElement)
		c.currentSize -= oldElem.item.Size
		oldElem.item = item
		c.currentSize += item.Size
		c.evictList.MoveToFront(elem)
	} else {
		elem := c.evictList.PushFront(&mediaElement{key: id, item: item})
		c.items[id] = elem
		c.currentSize += item.Size
	}

	for c.currentSize > c.maxBytes && c.evictList.Len() > 0 {
		oldest := c.evictList.Back()
		if oldest != nil {
			c.removeElement(oldest)
		}
	}
}

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

func (c *MediaCache) CurrentSize() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentSize
}

func (c *MediaCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func (c *MediaCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element)
	c.evictList.Init()
	c.currentSize = 0
}
