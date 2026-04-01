// Package cache provides an in-memory cache implementation with Redis-compatible operations.
//
// The cache supports multiple data types including strings, hashes, lists, and sets,
// with automatic expiration and thread-safe operations. It's designed to be the core
// storage engine for the CacheMir distributed caching system.
//
// Supported Data Types:
//   - Strings: Simple key-value pairs with optional TTL
//   - Hashes: Field-value mappings (like Redis hashes)
//   - Lists: Ordered collections with head/tail operations
//   - Sets: Unordered collections of unique members
//
// Example usage:
//
//	c := cache.New()
//	defer c.Close()
//
//	// String operations
//	c.Set("user:123", "john_doe", time.Hour)
//	value, exists := c.Get("user:123")
//
//	// Hash operations
//	c.HSet("user:123:profile", "name", "John Doe")
//	c.HSet("user:123:profile", "email", "john@example.com")
//	profile := c.HGetAll("user:123:profile")
//
//	// List operations
//	c.LPush("tasks", "task1", "task2", "task3")
//	task, exists := c.LPop("tasks")
//
//	// Set operations
//	c.SAdd("tags", "golang", "cache", "distributed")
//	members := c.SMembers("tags")
//
// All operations are thread-safe and can be called concurrently from multiple goroutines.
// The cache automatically handles expiration cleanup in the background.
package cache

import (
	"fmt"
	"maps"
	"strconv"
	"sync"
	"time"
)

// ValueType represents the type of data stored in a cache value.
type ValueType uint8

const (
	TypeString ValueType = iota // String value
	TypeHash                    // Hash value (map[string]string)
	TypeList                    // List value ([]string)
	TypeSet                     // Set value (map[string]struct{})
)

// value represents a single cache entry. The active field is determined by Type.
// Using a typed union avoids interface{} boxing and type assertion overhead.
type value struct {
	strVal    string
	hashVal   map[string]string
	listVal   []string
	setVal    map[string]struct{}
	expiresAt time.Time
	vtype     ValueType
}

func (v *value) isExpired(now time.Time) bool {
	return !v.expiresAt.IsZero() && now.After(v.expiresAt)
}

// Cache provides thread-safe in-memory storage with Redis-compatible operations.
// Close must be called to stop the background cleanup goroutine.
//
// Example:
//
//	c := cache.New()
//	defer c.Close()
//
//	c.Set("session:abc", "user123", 30*time.Minute)
//	if v, exists := c.Get("session:abc"); exists {
//		fmt.Printf("Session data: %s\n", v)
//	}
type Cache struct {
	data map[string]*value
	mu   sync.RWMutex
	stop chan struct{}
}

// New creates a new Cache instance and starts the background expiration cleanup.
// Call Close() to stop the background goroutine and free resources.
func New() *Cache {
	c := &Cache{
		data: make(map[string]*value),
		stop: make(chan struct{}),
	}
	go c.cleanupExpired()
	return c
}

// Close stops the background cleanup goroutine. It is safe to call multiple times.
func (c *Cache) Close() {
	select {
	case <-c.stop:
		// already closed
	default:
		close(c.stop)
	}
}

// cleanupExpired runs in the background to remove expired keys every minute.
func (c *Cache) cleanupExpired() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			c.mu.Lock()
			for key, v := range c.data {
				if v.isExpired(now) {
					delete(c.data, key)
				}
			}
			c.mu.Unlock()
		case <-c.stop:
			return
		}
	}
}

// Get retrieves a string value from the cache.
// Returns the value and true if the key exists and hasn't expired.
func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	v, exists := c.data[key]
	c.mu.RUnlock()

	if !exists || v.isExpired(time.Now()) || v.vtype != TypeString {
		return "", false
	}
	return v.strVal, true
}

// Set stores a string value in the cache with an optional TTL.
// If TTL is 0, the key will not expire.
func (c *Cache) Set(key, val string, ttl time.Duration) {
	v := &value{
		vtype:  TypeString,
		strVal: val,
	}
	if ttl > 0 {
		v.expiresAt = time.Now().Add(ttl)
	}

	c.mu.Lock()
	c.data[key] = v
	c.mu.Unlock()
}

// Del removes a key from the cache.
// Returns true if the key existed and was deleted.
func (c *Cache) Del(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.data[key]; exists {
		delete(c.data, key)
		return true
	}
	return false
}

// Exists checks if a key exists in the cache and hasn't expired.
func (c *Cache) Exists(key string) bool {
	c.mu.RLock()
	v, exists := c.data[key]
	c.mu.RUnlock()

	return exists && !v.isExpired(time.Now())
}

// Incr increments the integer value of a key by 1.
// If the key doesn't exist, it's initialized to 1.
func (c *Cache) Incr(key string) (int64, error) {
	return c.IncrBy(key, 1)
}

// Decr decrements the integer value of a key by 1.
// If the key doesn't exist, it's initialized to -1.
func (c *Cache) Decr(key string) (int64, error) {
	return c.IncrBy(key, -1)
}

// IncrBy increments the integer value of a key by delta.
// If the key doesn't exist, it's initialized to delta.
func (c *Cache) IncrBy(key string, delta int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, exists := c.data[key]
	if !exists || v.isExpired(time.Now()) {
		c.data[key] = &value{
			vtype:  TypeString,
			strVal: strconv.FormatInt(delta, 10),
		}
		return delta, nil
	}

	if v.vtype != TypeString {
		return 0, fmt.Errorf("value is not a string")
	}

	current, err := strconv.ParseInt(v.strVal, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("value is not an integer")
	}

	newVal := current + delta
	v.strVal = strconv.FormatInt(newVal, 10)
	return newVal, nil
}

// Expire sets a timeout on a key. Returns true if the expiration was set.
func (c *Cache) Expire(key string, ttl time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, exists := c.data[key]
	if !exists || v.isExpired(time.Now()) {
		return false
	}

	v.expiresAt = time.Now().Add(ttl)
	return true
}

// TTL returns the remaining time to live of a key.
// Returns -2s if the key doesn't exist, -1s if it has no expiration.
func (c *Cache) TTL(key string) time.Duration {
	c.mu.RLock()
	v, exists := c.data[key]
	c.mu.RUnlock()

	if !exists || v.isExpired(time.Now()) {
		return -2 * time.Second
	}
	if v.expiresAt.IsZero() {
		return -1 * time.Second
	}
	remaining := time.Until(v.expiresAt)
	if remaining <= 0 {
		return -2 * time.Second
	}
	return remaining
}

// Persist removes the expiration from a key, making it permanent.
// Returns true if the expiration was removed.
func (c *Cache) Persist(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, exists := c.data[key]
	if !exists || v.isExpired(time.Now()) {
		return false
	}

	v.expiresAt = time.Time{}
	return true
}

// HGet retrieves the value of a hash field.
func (c *Cache) HGet(key, field string) (string, bool) {
	c.mu.RLock()
	v, exists := c.data[key]
	c.mu.RUnlock()

	if !exists || v.isExpired(time.Now()) || v.vtype != TypeHash {
		return "", false
	}
	val, ok := v.hashVal[field]
	return val, ok
}

// HSet sets the value of a hash field, creating the hash if it doesn't exist.
func (c *Cache) HSet(key, field, val string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, exists := c.data[key]
	if !exists || v.isExpired(time.Now()) {
		v = &value{
			vtype:   TypeHash,
			hashVal: make(map[string]string),
		}
		c.data[key] = v
	} else if v.vtype != TypeHash {
		return
	}

	v.hashVal[field] = val
}

// HDel deletes a field from a hash. Returns true if the field existed.
func (c *Cache) HDel(key, field string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, exists := c.data[key]
	if !exists || v.isExpired(time.Now()) || v.vtype != TypeHash {
		return false
	}

	if _, ok := v.hashVal[field]; ok {
		delete(v.hashVal, field)
		return true
	}
	return false
}

// HExists checks if a field exists in a hash.
func (c *Cache) HExists(key, field string) bool {
	c.mu.RLock()
	v, exists := c.data[key]
	c.mu.RUnlock()

	if !exists || v.isExpired(time.Now()) || v.vtype != TypeHash {
		return false
	}
	_, ok := v.hashVal[field]
	return ok
}

// HGetAll returns all fields and values in a hash as a copy.
func (c *Cache) HGetAll(key string) map[string]string {
	c.mu.RLock()
	v, exists := c.data[key]
	c.mu.RUnlock()

	if !exists || v.isExpired(time.Now()) || v.vtype != TypeHash {
		return map[string]string{}
	}

	result := make(map[string]string, len(v.hashVal))
	maps.Copy(result, v.hashVal)
	return result
}

// LPush inserts values at the head of a list.
// Values are inserted in reverse argument order so the last argument ends up at the front.
// Returns the new length of the list.
func (c *Cache) LPush(key string, values ...string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, exists := c.data[key]
	if !exists || v.isExpired(time.Now()) {
		v = &value{
			vtype:   TypeList,
			listVal: make([]string, 0, len(values)),
		}
		c.data[key] = v
	} else if v.vtype != TypeList {
		return 0
	}

	// Prepend efficiently: allocate new slice with capacity for all elements.
	newList := make([]string, len(values)+len(v.listVal))
	// Copy values in reverse order to the front.
	for i, val := range values {
		newList[len(values)-1-i] = val
	}
	copy(newList[len(values):], v.listVal)
	v.listVal = newList
	return len(newList)
}

// RPush inserts values at the tail of a list. Returns the new length.
func (c *Cache) RPush(key string, values ...string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, exists := c.data[key]
	if !exists || v.isExpired(time.Now()) {
		v = &value{
			vtype:   TypeList,
			listVal: make([]string, 0, len(values)),
		}
		c.data[key] = v
	} else if v.vtype != TypeList {
		return 0
	}

	v.listVal = append(v.listVal, values...)
	return len(v.listVal)
}

// LPop removes and returns the first element from a list.
func (c *Cache) LPop(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, exists := c.data[key]
	if !exists || v.isExpired(time.Now()) || v.vtype != TypeList || len(v.listVal) == 0 {
		return "", false
	}

	result := v.listVal[0]
	v.listVal = v.listVal[1:]
	return result, true
}

// RPop removes and returns the last element from a list.
func (c *Cache) RPop(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, exists := c.data[key]
	if !exists || v.isExpired(time.Now()) || v.vtype != TypeList || len(v.listVal) == 0 {
		return "", false
	}

	last := len(v.listVal) - 1
	result := v.listVal[last]
	v.listVal = v.listVal[:last]
	return result, true
}

// LLen returns the length of a list.
func (c *Cache) LLen(key string) int {
	c.mu.RLock()
	v, exists := c.data[key]
	c.mu.RUnlock()

	if !exists || v.isExpired(time.Now()) || v.vtype != TypeList {
		return 0
	}
	return len(v.listVal)
}

// SAdd adds members to a set. Returns the number of new members added.
func (c *Cache) SAdd(key string, members ...string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, exists := c.data[key]
	if !exists || v.isExpired(time.Now()) {
		v = &value{
			vtype:  TypeSet,
			setVal: make(map[string]struct{}, len(members)),
		}
		c.data[key] = v
	} else if v.vtype != TypeSet {
		return 0
	}

	added := 0
	for _, member := range members {
		if _, ok := v.setVal[member]; !ok {
			v.setVal[member] = struct{}{}
			added++
		}
	}
	return added
}

// SRem removes members from a set. Returns the number of members removed.
func (c *Cache) SRem(key string, members ...string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, exists := c.data[key]
	if !exists || v.isExpired(time.Now()) || v.vtype != TypeSet {
		return 0
	}

	removed := 0
	for _, member := range members {
		if _, ok := v.setVal[member]; ok {
			delete(v.setVal, member)
			removed++
		}
	}
	return removed
}

// SMembers returns all members of a set as a slice.
func (c *Cache) SMembers(key string) []string {
	c.mu.RLock()
	v, exists := c.data[key]
	c.mu.RUnlock()

	if !exists || v.isExpired(time.Now()) || v.vtype != TypeSet {
		return []string{}
	}

	members := make([]string, 0, len(v.setVal))
	for member := range v.setVal {
		members = append(members, member)
	}
	return members
}

// SIsMember checks if a member exists in a set.
func (c *Cache) SIsMember(key, member string) bool {
	c.mu.RLock()
	v, exists := c.data[key]
	c.mu.RUnlock()

	if !exists || v.isExpired(time.Now()) || v.vtype != TypeSet {
		return false
	}
	_, ok := v.setVal[member]
	return ok
}

// Stats returns statistics about the current cache state.
func (c *Cache) Stats() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	typeCount := make(map[string]int, 4)
	expiredCount := 0
	now := time.Now()

	for _, v := range c.data {
		switch v.vtype {
		case TypeString:
			typeCount["string"]++
		case TypeHash:
			typeCount["hash"]++
		case TypeList:
			typeCount["list"]++
		case TypeSet:
			typeCount["set"]++
		}
		if v.isExpired(now) {
			expiredCount++
		}
	}

	return map[string]any{
		"keys":    len(c.data),
		"types":   typeCount,
		"expired": expiredCount,
	}
}
