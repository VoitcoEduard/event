package cache

import (
	"testing"
	"time"
)

func TestCacheGetSet(t *testing.T) {
	c := New(time.Hour)
	c.Set("k", []byte("v"))

	got, ok := c.Get("k")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != "v" {
		t.Fatalf("got %q, want %q", got, "v")
	}
}

func TestCacheExpiry(t *testing.T) {
	c := New(50 * time.Millisecond)
	c.Set("k", []byte("v"))

	time.Sleep(80 * time.Millisecond)

	if _, ok := c.Get("k"); ok {
		t.Fatal("expected cache miss after TTL")
	}
}

func TestCacheDelete(t *testing.T) {
	c := New(time.Hour)
	c.Set("k", []byte("v"))
	c.Delete("k")

	if _, ok := c.Get("k"); ok {
		t.Fatal("expected cache miss after delete")
	}
}

func TestCacheDeletePrefix(t *testing.T) {
	c := New(time.Hour)
	c.Set("evt:1|u:a", []byte("a"))
	c.Set("evt:1|u:b", []byte("b"))
	c.Set("evt:2|u:a", []byte("c"))
	c.Set("evt:list", []byte("l"))

	c.DeletePrefix("evt:1|")

	for _, key := range []string{"evt:1|u:a", "evt:1|u:b"} {
		if _, ok := c.Get(key); ok {
			t.Fatalf("expected miss for %s after DeletePrefix", key)
		}
	}
	if _, ok := c.Get("evt:2|u:a"); !ok {
		t.Fatal("expected hit for untouched key evt:2|u:a")
	}
	if _, ok := c.Get("evt:list"); !ok {
		t.Fatal("expected hit for untouched key evt:list")
	}
}

func TestCacheOverflowEvicts(t *testing.T) {
	c := &Cache{
		entries:    make(map[string]entry),
		ttl:        time.Hour,
		maxEntries: 2,
	}
	c.Set("a", []byte("1"))
	c.Set("b", []byte("2"))
	c.Set("c", []byte("3"))

	if len(c.entries) > 2 {
		t.Fatalf("expected at most 2 entries, got %d", len(c.entries))
	}
}
