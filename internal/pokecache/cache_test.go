package pokecache

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	cache := NewCache(5 * time.Second)

	cache.Add("key", []byte("value"))
	val, found := cache.Get("key")
	if !found || string(val) != "value" {
		t.Errorf("expected to find value")
	}
	_, found = cache.Get("nonexistent")
	if found {
		t.Errorf("should not have found nonexistent key")
	}
}
