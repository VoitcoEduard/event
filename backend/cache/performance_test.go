package cache

import (
	"testing"
	"time"
)

func BenchmarkEventCacheHitParallel(b *testing.B) {
	c := New(time.Hour)
	c.Set("event|player", make([]byte, 16*1024))
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, ok := c.Get("event|player"); !ok {
				b.Fatal("unexpected cache miss")
			}
		}
	})
}

func BenchmarkMediaCacheHitParallel(b *testing.B) {
	c := NewMediaCache(64 * 1024 * 1024)
	c.Set("image", MediaItem{ID: "image", Data: make([]byte, 512*1024), Size: 512 * 1024})
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, ok := c.Get("image"); !ok {
				b.Fatal("unexpected media cache miss")
			}
		}
	})
}
