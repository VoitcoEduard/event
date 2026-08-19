package cache

import (
	"testing"
	"time"
)

func TestMediaCacheGetSet(t *testing.T) {
	c := NewMediaCache(1024 * 1024) // 1MB

	item := MediaItem{
		ID:        "media-1",
		Filename:  "test.jpg",
		MimeType:  "image/jpeg",
		Data:      []byte("fake-jpeg-data"),
		CreatedAt: time.Now(),
		Size:      int64(len("fake-jpeg-data")),
	}

	c.Set(item.ID, item)

	got, found := c.Get("media-1")
	if !found {
		t.Fatalf("expected media-1 to be found in cache")
	}
	if string(got.Data) != "fake-jpeg-data" {
		t.Errorf("expected data 'fake-jpeg-data', got '%s'", string(got.Data))
	}

	c.Delete("media-1")
	_, found = c.Get("media-1")
	if found {
		t.Errorf("expected media-1 to be deleted")
	}
}

func TestMediaCacheLRUEviction(t *testing.T) {
	// Cache capacity: 100 bytes
	c := NewMediaCache(100)

	item1 := MediaItem{ID: "1", Data: make([]byte, 30), Size: 30}
	item2 := MediaItem{ID: "2", Data: make([]byte, 30), Size: 30}
	item3 := MediaItem{ID: "3", Data: make([]byte, 30), Size: 30}

	c.Set("1", item1)
	c.Set("2", item2)
	c.Set("3", item3)

	if c.Len() != 3 {
		t.Fatalf("expected 3 items, got %d", c.Len())
	}

	// Access item1 to make it most recently used (order now: 1, 3, 2)
	c.Get("1")

	// Add item4 (30 bytes). Total would be 120 bytes > 100 bytes -> evict oldest (item 2)
	item4 := MediaItem{ID: "4", Data: make([]byte, 30), Size: 30}
	c.Set("4", item4)

	if _, found := c.Get("2"); found {
		t.Errorf("expected item 2 to be evicted")
	}
	if _, found := c.Get("1"); !found {
		t.Errorf("expected item 1 to remain in cache")
	}
	if _, found := c.Get("3"); !found {
		t.Errorf("expected item 3 to remain in cache")
	}
	if _, found := c.Get("4"); !found {
		t.Errorf("expected item 4 to remain in cache")
	}
}
