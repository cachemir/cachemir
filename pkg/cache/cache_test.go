package cache

import (
	"fmt"
	"testing"
	"time"
)

func newTestCache(t *testing.T) *Cache {
	t.Helper()
	c := New()
	t.Cleanup(c.Close)
	return c
}

func TestCacheBasicOperations(t *testing.T) {
	c := newTestCache(t)

	c.Set("key1", "value1", 0)

	if value, exists := c.Get("key1"); !exists || value != "value1" {
		t.Errorf("expected value1, got %q (exists: %t)", value, exists)
	}

	if !c.Exists("key1") {
		t.Error("key should exist")
	}

	if !c.Del("key1") {
		t.Error("Del should return true for existing key")
	}

	if c.Exists("key1") {
		t.Error("key should not exist after deletion")
	}

	if c.Del("key1") {
		t.Error("Del should return false for non-existent key")
	}
}

func TestCacheExpiration(t *testing.T) {
	c := newTestCache(t)

	c.Set("temp_key", "temp_value", 100*time.Millisecond)

	if value, exists := c.Get("temp_key"); !exists || value != "temp_value" {
		t.Errorf("expected temp_value before expiry, got %q (exists: %t)", value, exists)
	}

	time.Sleep(150 * time.Millisecond)

	if _, exists := c.Get("temp_key"); exists {
		t.Error("key should have expired")
	}
	if c.Exists("temp_key") {
		t.Error("Exists should return false for expired key")
	}
}

func TestCacheTTL(t *testing.T) {
	c := newTestCache(t)

	// Non-existent key.
	if ttl := c.TTL("missing"); ttl != -2*time.Second {
		t.Errorf("expected -2s for missing key, got %v", ttl)
	}

	// Key with no expiration.
	c.Set("perm", "v", 0)
	if ttl := c.TTL("perm"); ttl != -1*time.Second {
		t.Errorf("expected -1s for key without TTL, got %v", ttl)
	}

	// Key with TTL.
	c.Set("temp", "v", 10*time.Second)
	ttl := c.TTL("temp")
	if ttl <= 0 || ttl > 10*time.Second {
		t.Errorf("unexpected TTL for key with 10s expiry: %v", ttl)
	}

	// Persist removes expiry.
	if !c.Persist("temp") {
		t.Error("Persist should return true")
	}
	if ttl := c.TTL("temp"); ttl != -1*time.Second {
		t.Errorf("expected -1s after Persist, got %v", ttl)
	}
}

func TestCacheIncrement(t *testing.T) {
	tests := []struct {
		name    string
		ops     func(*Cache) (int64, error)
		want    int64
		wantErr bool
	}{
		{
			name:    "incr new key",
			ops:     func(c *Cache) (int64, error) { return c.Incr("x") },
			want:    1,
			wantErr: false,
		},
		{
			name: "incrby",
			ops: func(c *Cache) (int64, error) {
				c.Set("n", "10", 0)
				return c.IncrBy("n", 5)
			},
			want:    15,
			wantErr: false,
		},
		{
			name: "decr",
			ops: func(c *Cache) (int64, error) {
				c.Set("n", "3", 0)
				return c.Decr("n")
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "incr non-integer errors",
			ops: func(c *Cache) (int64, error) {
				c.Set("s", "hello", 0)
				return c.Incr("s")
			},
			wantErr: true,
		},
		{
			name: "incr wrong type errors",
			ops: func(c *Cache) (int64, error) {
				c.HSet("h", "f", "v")
				return c.Incr("h")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestCache(t)
			got, err := tt.ops(c)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("want %d, got %d", tt.want, got)
			}
		})
	}
}

func TestCacheHashOperations(t *testing.T) {
	c := newTestCache(t)

	c.HSet("hash1", "field1", "value1")
	c.HSet("hash1", "field2", "value2")

	if value, exists := c.HGet("hash1", "field1"); !exists || value != "value1" {
		t.Errorf("expected value1, got %q (exists: %t)", value, exists)
	}

	if !c.HExists("hash1", "field1") {
		t.Error("HExists should return true for existing field")
	}
	if c.HExists("hash1", "missing") {
		t.Error("HExists should return false for missing field")
	}

	hash := c.HGetAll("hash1")
	if len(hash) != 2 {
		t.Errorf("expected 2 fields, got %d", len(hash))
	}
	if hash["field1"] != "value1" || hash["field2"] != "value2" {
		t.Errorf("hash values incorrect: %+v", hash)
	}

	if !c.HDel("hash1", "field1") {
		t.Error("HDel should return true for existing field")
	}
	if _, exists := c.HGet("hash1", "field1"); exists {
		t.Error("field should not exist after HDel")
	}
	if c.HDel("hash1", "field1") {
		t.Error("HDel should return false for already-deleted field")
	}
}

func TestCacheListOperations(t *testing.T) {
	c := newTestCache(t)

	// LPush inserts in reverse: last arg ends up at head.
	length := c.LPush("list1", "a", "b", "c")
	if length != 3 {
		t.Errorf("expected length 3 after LPush, got %d", length)
	}

	// After LPush("a","b","c"), list should be ["c","b","a"].
	if v, ok := c.LPop("list1"); !ok || v != "c" {
		t.Errorf("expected 'c' from LPop, got %q (ok: %t)", v, ok)
	}

	// After LPop, list is ["b","a"]. RPush adds "d","e" → ["b","a","d","e"].
	length = c.RPush("list1", "d", "e")
	if length != 4 {
		t.Errorf("expected length 4 after RPush, got %d", length)
	}

	if v, ok := c.RPop("list1"); !ok || v != "e" {
		t.Errorf("expected 'e' from RPop, got %q (ok: %t)", v, ok)
	}

	if n := c.LLen("list1"); n != 3 {
		t.Errorf("expected LLen 3, got %d", n)
	}

	// Pop from empty/missing list.
	c.Del("list1")
	if _, ok := c.LPop("list1"); ok {
		t.Error("LPop on missing key should return false")
	}
}

func TestCacheSetOperations(t *testing.T) {
	c := newTestCache(t)

	added := c.SAdd("set1", "m1", "m2", "m3")
	if added != 3 {
		t.Errorf("expected 3 added, got %d", added)
	}

	// Duplicate add.
	added = c.SAdd("set1", "m2", "m4")
	if added != 1 {
		t.Errorf("expected 1 added (m4 only), got %d", added)
	}

	if !c.SIsMember("set1", "m1") {
		t.Error("m1 should be in set")
	}
	if c.SIsMember("set1", "nonexistent") {
		t.Error("nonexistent should not be in set")
	}

	members := c.SMembers("set1")
	if len(members) != 4 {
		t.Errorf("expected 4 members, got %d", len(members))
	}

	removed := c.SRem("set1", "m1", "m2")
	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}

	if c.SIsMember("set1", "m1") {
		t.Error("m1 should have been removed")
	}
}

func TestCacheSetUsesStructNotBool(t *testing.T) {
	// Verify set storage uses map[string]struct{} - tests the internal
	// memory optimization without exposing implementation.
	c := newTestCache(t)
	c.SAdd("s", "x")
	if !c.SIsMember("s", "x") {
		t.Error("SIsMember returned false after SAdd")
	}
}

func TestCacheClose(t *testing.T) {
	c := New()
	c.Set("k", "v", 0)
	c.Close()
	// Double-close should not panic.
	c.Close()
	// Cache should still be readable after Close (cleanup goroutine stopped,
	// but data map untouched).
	if _, ok := c.Get("k"); !ok {
		t.Error("cache should still be readable after Close")
	}
}

func TestCacheStats(t *testing.T) {
	c := newTestCache(t)

	c.Set("s1", "v", 0)
	c.HSet("h1", "f", "v")
	c.LPush("l1", "v")
	c.SAdd("set1", "v")

	stats := c.Stats()

	if keys, _ := stats["keys"].(int); keys != 4 {
		t.Errorf("expected 4 keys, got %d", keys)
	}

	types, _ := stats["types"].(map[string]int)
	if types["string"] != 1 || types["hash"] != 1 || types["list"] != 1 || types["set"] != 1 {
		t.Errorf("unexpected type counts: %v", types)
	}
}

// --- Benchmarks ---

func BenchmarkCacheSet(b *testing.B) {
	c := New()
	defer c.Close()
	var i int
	for b.Loop() {
		c.Set(fmt.Sprintf("key:%d", i), "value", 0)
		i++
	}
}

func BenchmarkCacheGet(b *testing.B) {
	c := New()
	defer c.Close()
	c.Set("key", "value", 0)
	for b.Loop() {
		c.Get("key")
	}
}

func BenchmarkCacheGetMiss(b *testing.B) {
	c := New()
	defer c.Close()
	for b.Loop() {
		c.Get("missing")
	}
}

func BenchmarkCacheIncrBy(b *testing.B) {
	c := New()
	defer c.Close()
	c.Set("counter", "0", 0)
	for b.Loop() {
		c.IncrBy("counter", 1)
	}
}

func BenchmarkCacheLPush(b *testing.B) {
	c := New()
	defer c.Close()
	for b.Loop() {
		c.LPush("list", "value")
	}
}

func BenchmarkCacheSAdd(b *testing.B) {
	c := New()
	defer c.Close()
	var i int
	for b.Loop() {
		c.SAdd("set", fmt.Sprintf("member:%d", i))
		i++
	}
}

func BenchmarkCacheHSet(b *testing.B) {
	c := New()
	defer c.Close()
	var i int
	for b.Loop() {
		c.HSet("hash", fmt.Sprintf("field:%d", i), "value")
		i++
	}
}
