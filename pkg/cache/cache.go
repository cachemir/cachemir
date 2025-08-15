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
//	cache := cache.New()
//
//	// String operations
//	cache.Set("user:123", "john_doe", time.Hour)
//	value, exists := cache.Get("user:123")
//
//	// Hash operations
//	cache.HSet("user:123:profile", "name", "John Doe")
//	cache.HSet("user:123:profile", "email", "john@example.com")
//	profile := cache.HGetAll("user:123:profile")
//
//	// List operations
//	cache.LPush("tasks", "task1", "task2", "task3")
//	task, exists := cache.LPop("tasks")
//
//	// Set operations
//	cache.SAdd("tags", "golang", "cache", "distributed")
//	members := cache.SMembers("tags")
//
// All operations are thread-safe and can be called concurrently from multiple goroutines.
// The cache automatically handles expiration cleanup in the background.
package cache

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

// ValueType represents the type of data stored in a cache value.
// Different types support different operations and have different storage formats.
type ValueType uint8

const (
	TypeString ValueType = iota // String value ([]byte)
	TypeHash                    // Hash value (map[string]string)
	TypeList                    // List value ([]string)
	TypeSet                     // Set value (map[string]bool)
)

// Value represents a single cache entry with its data, type, and expiration.
// The Data field contains the actual value, which varies by type:
//   - TypeString: string
//   - TypeHash: map[string]string
//   - TypeList: []string
//   - TypeSet: map[string]bool
type Value struct {
	Data      interface{} // The actual data (type depends on Type field)
	ExpiresAt time.Time   // When this value expires (zero means no expiration)
	Type      ValueType   // The type of data stored
}

// Cache provides thread-safe in-memory storage with Redis-compatible operations.
// It supports automatic expiration, multiple data types, and concurrent access.
// The cache runs a background goroutine to clean up expired keys.
//
// Example:
//
//	cache := cache.New()
//	defer cache.Close() // Stop background cleanup (if implemented)
//
//	cache.Set("session:abc", "user123", 30*time.Minute)
//	if value, exists := cache.Get("session:abc"); exists {
//		fmt.Printf("Session data: %s\n", value)
//	}
type Cache struct {
	data map[string]*Value // The actual cache storage
	mu   sync.RWMutex      // Protects the data map
}

// New creates a new Cache instance and starts the background expiration cleanup.
// The cleanup goroutine runs every minute to remove expired keys.
//
// Example:
//
//	cache := cache.New()
//	// Cache is ready to use
//
// Returns:
//   - A new Cache instance ready for use
func New() *Cache {
	c := &Cache{
		data: make(map[string]*Value),
	}
	go c.cleanupExpired()
	return c
}

// cleanupExpired runs in a background goroutine to periodically remove expired keys.
// It runs every minute and removes all keys that have passed their expiration time.
// This prevents memory leaks from expired but unaccessed keys.
func (c *Cache) cleanupExpired() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, value := range c.data {
			if !value.ExpiresAt.IsZero() && now.After(value.ExpiresAt) {
				delete(c.data, key)
			}
		}
		c.mu.Unlock()
	}
}

// isExpired checks if a value has expired based on the current time.
// Returns true if the value has an expiration time and it has passed.
func (c *Cache) isExpired(value *Value) bool {
	return !value.ExpiresAt.IsZero() && time.Now().After(value.ExpiresAt)
}

// Get retrieves a string value from the cache.
// Returns the value and true if the key exists and hasn't expired.
// Returns empty string and false if the key doesn't exist, has expired, or is not a string.
//
// Example:
//
//	cache.Set("greeting", "Hello, World!", 0)
//	if value, exists := cache.Get("greeting"); exists {
//		fmt.Printf("Greeting: %s\n", value)
//	}
//
// Parameters:
//   - key: The key to retrieve
//
// Returns:
//   - The string value if found
//   - Boolean indicating if the key exists and is valid
func (c *Cache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) {
		return "", false
	}

	if value.Type != TypeString {
		return "", false
	}

	if str, ok := value.Data.(string); ok {
		return str, true
	}
	return "", false
}

// Set stores a string value in the cache with an optional TTL.
// If TTL is 0, the key will not expire. If TTL is positive, the key
// will expire after the specified duration.
//
// Example:
//
//	// Set without expiration
//	cache.Set("permanent", "value", 0)
//
//	// Set with 1 hour expiration
//	cache.Set("temporary", "value", time.Hour)
//
// Parameters:
//   - key: The key to store
//   - val: The string value to store
//   - ttl: Time-to-live duration (0 for no expiration)
func (c *Cache) Set(key, val string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	value := &Value{
		Type: TypeString,
		Data: val,
	}

	if ttl > 0 {
		value.ExpiresAt = time.Now().Add(ttl)
	}

	c.data[key] = value
}

// Del removes a key from the cache.
// Returns true if the key existed and was deleted, false otherwise.
//
// Example:
//
//	cache.Set("temp", "value", 0)
//	if cache.Del("temp") {
//		fmt.Println("Key deleted successfully")
//	}
//
// Parameters:
//   - key: The key to delete
//
// Returns:
//   - Boolean indicating if the key was deleted
func (c *Cache) Del(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, exists := c.data[key]
	if exists {
		delete(c.data, key)
		return true
	}
	return false
}

// Exists checks if a key exists in the cache and hasn't expired.
// Returns true if the key exists and is valid, false otherwise.
//
// Example:
//
//	cache.Set("check", "value", 0)
//	if cache.Exists("check") {
//		fmt.Println("Key exists")
//	}
//
// Parameters:
//   - key: The key to check
//
// Returns:
//   - Boolean indicating if the key exists and is valid
func (c *Cache) Exists(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) {
		return false
	}
	return true
}

// Incr increments the integer value of a key by 1.
// If the key doesn't exist, it's set to 1.
// If the key exists but is not a valid integer, an error is returned.
//
// Example:
//
//	count, err := cache.Incr("page_views")
//	if err != nil {
//		log.Printf("Error: %v", err)
//	} else {
//		fmt.Printf("Page views: %d\n", count)
//	}
//
// Parameters:
//   - key: The key to increment
//
// Returns:
//   - The new integer value after incrementing
//   - Error if the key exists but is not an integer
func (c *Cache) Incr(key string) (int64, error) {
	return c.IncrBy(key, 1)
}

// Decr decrements the integer value of a key by 1.
// If the key doesn't exist, it's set to -1.
// If the key exists but is not a valid integer, an error is returned.
//
// Example:
//
//	count, err := cache.Decr("countdown")
//	if err != nil {
//		log.Printf("Error: %v", err)
//	} else {
//		fmt.Printf("Countdown: %d\n", count)
//	}
//
// Parameters:
//   - key: The key to decrement
//
// Returns:
//   - The new integer value after decrementing
//   - Error if the key exists but is not an integer
func (c *Cache) Decr(key string) (int64, error) {
	return c.IncrBy(key, -1)
}

// IncrBy increments the integer value of a key by the specified delta.
// If the key doesn't exist, it's set to the delta value.
// If the key exists but is not a valid integer, an error is returned.
//
// Example:
//
//	// Increment by 5
//	count, err := cache.IncrBy("score", 5)
//
//	// Decrement by 3 (negative delta)
//	count, err = cache.IncrBy("score", -3)
//
// Parameters:
//   - key: The key to modify
//   - delta: The amount to add (can be negative)
//
// Returns:
//   - The new integer value after the operation
//   - Error if the key exists but is not an integer
func (c *Cache) IncrBy(key string, delta int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) {
		newValue := &Value{
			Type: TypeString,
			Data: strconv.FormatInt(delta, 10),
		}
		c.data[key] = newValue
		return delta, nil
	}

	if value.Type != TypeString {
		return 0, fmt.Errorf("value is not a string")
	}

	str, ok := value.Data.(string)
	if !ok {
		return 0, fmt.Errorf("value is not a string")
	}
	current, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("value is not an integer")
	}

	newVal := current + delta
	value.Data = strconv.FormatInt(newVal, 10)
	return newVal, nil
}

// Expire sets a timeout on a key. After the timeout, the key will be automatically deleted.
// Returns true if the timeout was set, false if the key doesn't exist or has already expired.
//
// Example:
//
//	cache.Set("temp", "value", 0) // No expiration initially
//	if cache.Expire("temp", 30*time.Second) {
//		fmt.Println("Key will expire in 30 seconds")
//	}
//
// Parameters:
//   - key: The key to set expiration on
//   - ttl: Time-to-live duration
//
// Returns:
//   - Boolean indicating if the expiration was set
func (c *Cache) Expire(key string, ttl time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) {
		return false
	}

	value.ExpiresAt = time.Now().Add(ttl)
	return true
}

// TTL returns the remaining time to live of a key.
// Returns the duration until expiration, or special values:
//   - Negative duration < -1 second: key doesn't exist
//   - -1 second: key exists but has no expiration
//   - 0 or positive: remaining time until expiration
//
// Example:
//
//	cache.Set("temp", "value", time.Minute)
//	ttl := cache.TTL("temp")
//	if ttl > 0 {
//		fmt.Printf("Key expires in %v\n", ttl)
//	}
//
// Parameters:
//   - key: The key to check
//
// Returns:
//   - Remaining time to live, or special negative values
func (c *Cache) TTL(key string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) {
		return -2 * time.Second
	}

	if value.ExpiresAt.IsZero() {
		return -1 * time.Second
	}

	remaining := time.Until(value.ExpiresAt)
	if remaining <= 0 {
		return -2 * time.Second
	}

	return remaining
}

// Persist removes the expiration from a key, making it permanent.
// Returns true if the expiration was removed, false if the key doesn't exist or has already expired.
//
// Example:
//
//	cache.Set("temp", "value", time.Minute)
//	if cache.Persist("temp") {
//		fmt.Println("Key is now permanent")
//	}
//
// Parameters:
//   - key: The key to make permanent
//
// Returns:
//   - Boolean indicating if the expiration was removed
func (c *Cache) Persist(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) {
		return false
	}

	value.ExpiresAt = time.Time{}
	return true
}

// HGet retrieves the value of a hash field.
// Returns the field value and true if the hash and field exist.
// Returns empty string and false if the hash doesn't exist, has expired, or the field doesn't exist.
//
// Example:
//
//	cache.HSet("user:123", "name", "John Doe")
//	if name, exists := cache.HGet("user:123", "name"); exists {
//		fmt.Printf("User name: %s\n", name)
//	}
//
// Parameters:
//   - key: The hash key
//   - field: The field name within the hash
//
// Returns:
//   - The field value if found
//   - Boolean indicating if the field exists
func (c *Cache) HGet(key, field string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) || value.Type != TypeHash {
		return "", false
	}

	hash, ok := value.Data.(map[string]string)
	if !ok {
		return "", false
	}
	val, exists := hash[field]
	return val, exists
}

// HSet sets the value of a hash field.
// If the hash doesn't exist, it's created. If the field exists, its value is updated.
//
// Example:
//
//	cache.HSet("user:123", "name", "John Doe")
//	cache.HSet("user:123", "email", "john@example.com")
//	cache.HSet("user:123", "age", "30")
//
// Parameters:
//   - key: The hash key
//   - field: The field name within the hash
//   - val: The field value to set
func (c *Cache) HSet(key, field, val string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) {
		value = &Value{
			Type: TypeHash,
			Data: make(map[string]string),
		}
		c.data[key] = value
	} else if value.Type != TypeHash {
		return
	}

	hash, ok := value.Data.(map[string]string)
	if !ok {
		return
	}
	hash[field] = val
}

// HDel deletes a field from a hash.
// Returns true if the field existed and was deleted, false otherwise.
//
// Example:
//
//	cache.HSet("user:123", "temp_field", "temp_value")
//	if cache.HDel("user:123", "temp_field") {
//		fmt.Println("Field deleted successfully")
//	}
//
// Parameters:
//   - key: The hash key
//   - field: The field name to delete
//
// Returns:
//   - Boolean indicating if the field was deleted
func (c *Cache) HDel(key, field string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) || value.Type != TypeHash {
		return false
	}

	hash, ok := value.Data.(map[string]string)
	if !ok {
		return false
	}
	_, exists = hash[field]
	if exists {
		delete(hash, field)
		return true
	}
	return false
}

// HExists checks if a field exists in a hash.
// Returns true if the field exists, false otherwise.
//
// Example:
//
//	cache.HSet("user:123", "name", "John")
//	exists := cache.HExists("user:123", "name") // returns true
//	exists = cache.HExists("user:123", "age")   // returns false
func (c *Cache) HExists(key, field string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) || value.Type != TypeHash {
		return false
	}

	hash, ok := value.Data.(map[string]string)
	if !ok {
		return false
	}
	_, exists = hash[field]
	return exists
}

// HGetAll returns all fields and values in a hash.
// Returns a map of field-value pairs. If the hash doesn't exist or has expired,
// returns an empty map.
//
// Example:
//
//	cache.HSet("user:123", "name", "John")
//	cache.HSet("user:123", "age", "30")
//	profile := cache.HGetAll("user:123")
//	for field, value := range profile {
//		fmt.Printf("%s: %s\n", field, value)
//	}
//
// Parameters:
//   - key: The hash key
//
// Returns:
//   - Map of all field-value pairs in the hash
func (c *Cache) HGetAll(key string) map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) || value.Type != TypeHash {
		return make(map[string]string)
	}

	hash, ok := value.Data.(map[string]string)
	if !ok {
		return make(map[string]string)
	}
	result := make(map[string]string, len(hash))
	for k, v := range hash {
		result[k] = v
	}
	return result
}

// LPush inserts values at the head (left) of a list.
// If the list doesn't exist, it's created. Values are inserted in reverse order,
// so the last value in the arguments becomes the first element in the list.
// Returns the new length of the list.
//
// Example:
//
//	// Creates list: ["c", "b", "a"]
//	length := cache.LPush("mylist", "a", "b", "c")
//	fmt.Printf("List length: %d\n", length)
//
// Parameters:
//   - key: The list key
//   - values: Values to insert at the head
//
// Returns:
//   - The new length of the list after insertion
func (c *Cache) LPush(key string, values ...string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) {
		value = &Value{
			Type: TypeList,
			Data: make([]string, 0),
		}
		c.data[key] = value
	} else if value.Type != TypeList {
		return 0
	}

	list, ok := value.Data.([]string)
	if !ok {
		return 0
	}
	for i := len(values) - 1; i >= 0; i-- {
		list = append([]string{values[i]}, list...)
	}
	value.Data = list
	return len(list)
}

// RPush inserts values at the tail (right) of a list.
// If the list doesn't exist, it's created. Values are appended in order.
// Returns the new length of the list.
//
// Example:
//
//	// Creates list: ["a", "b", "c"]
//	length := cache.RPush("mylist", "a", "b", "c")
//	fmt.Printf("List length: %d\n", length)
//
// Parameters:
//   - key: The list key
//   - values: Values to insert at the tail
//
// Returns:
//   - The new length of the list after insertion
func (c *Cache) RPush(key string, values ...string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) {
		value = &Value{
			Type: TypeList,
			Data: make([]string, 0),
		}
		c.data[key] = value
	} else if value.Type != TypeList {
		return 0
	}

	list, ok := value.Data.([]string)
	if !ok {
		return 0
	}
	list = append(list, values...)
	value.Data = list
	return len(list)
}

// LPop removes and returns the first element from the head (left) of a list.
// Returns the element and true if successful, empty string and false if the list
// doesn't exist, has expired, or is empty.
//
// Example:
//
//	cache.LPush("tasks", "task1", "task2", "task3")
//	if task, exists := cache.LPop("tasks"); exists {
//		fmt.Printf("Processing task: %s\n", task)
//	}
//
// Parameters:
//   - key: The list key
//
// Returns:
//   - The first element if successful
//   - Boolean indicating if an element was removed
func (c *Cache) LPop(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) || value.Type != TypeList {
		return "", false
	}

	list, ok := value.Data.([]string)
	if !ok {
		return "", false
	}
	if len(list) == 0 {
		return "", false
	}

	result := list[0]
	value.Data = list[1:]
	return result, true
}

// RPop removes and returns the last element from the tail (right) of a list.
// Returns the element and true if successful, empty string and false if the list
// doesn't exist, has expired, or is empty.
//
// Example:
//
//	cache.RPush("queue", "item1", "item2", "item3")
//	if item, exists := cache.RPop("queue"); exists {
//		fmt.Printf("Processing item: %s\n", item)
//	}
//
// Parameters:
//   - key: The list key
//
// Returns:
//   - The last element if successful
//   - Boolean indicating if an element was removed
func (c *Cache) RPop(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) || value.Type != TypeList {
		return "", false
	}

	list, ok := value.Data.([]string)
	if !ok {
		return "", false
	}
	if len(list) == 0 {
		return "", false
	}

	result := list[len(list)-1]
	value.Data = list[:len(list)-1]
	return result, true
}

// LLen returns the length of a list.
// Returns 0 if the list doesn't exist, has expired, or is not a list.
//
// Example:
//
//	cache.LPush("mylist", "a", "b", "c")
//	length := cache.LLen("mylist")
//	fmt.Printf("List has %d elements\n", length)
//
// Parameters:
//   - key: The list key
//
// Returns:
//   - The number of elements in the list
func (c *Cache) LLen(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) || value.Type != TypeList {
		return 0
	}

	list, ok := value.Data.([]string)
	if !ok {
		return 0
	}
	return len(list)
}

// SAdd adds members to a set.
// If the set doesn't exist, it's created. Duplicate members are ignored.
// Returns the number of members that were actually added (not counting duplicates).
//
// Example:
//
//	added := cache.SAdd("tags", "golang", "cache", "distributed", "golang")
//	fmt.Printf("Added %d new tags\n", added) // Prints 3, not 4
//
// Parameters:
//   - key: The set key
//   - members: Members to add to the set
//
// Returns:
//   - The number of members actually added (excluding duplicates)
func (c *Cache) SAdd(key string, members ...string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) {
		value = &Value{
			Type: TypeSet,
			Data: make(map[string]bool),
		}
		c.data[key] = value
	} else if value.Type != TypeSet {
		return 0
	}

	set, ok := value.Data.(map[string]bool)
	if !ok {
		return 0
	}
	added := 0
	for _, member := range members {
		if !set[member] {
			set[member] = true
			added++
		}
	}
	return added
}

// SRem removes members from a set.
// Returns the number of members that were actually removed.
//
// Example:
//
//	cache.SAdd("tags", "golang", "cache", "distributed")
//	removed := cache.SRem("tags", "cache", "nonexistent")
//	fmt.Printf("Removed %d tags\n", removed) // Prints 1
//
// Parameters:
//   - key: The set key
//   - members: Members to remove from the set
//
// Returns:
//   - The number of members actually removed
func (c *Cache) SRem(key string, members ...string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) || value.Type != TypeSet {
		return 0
	}

	set, ok := value.Data.(map[string]bool)
	if !ok {
		return 0
	}
	removed := 0
	for _, member := range members {
		if set[member] {
			delete(set, member)
			removed++
		}
	}
	return removed
}

// SMembers returns all members of a set.
// Returns an empty slice if the set doesn't exist, has expired, or is not a set.
// The order of members is not guaranteed.
//
// Example:
//
//	cache.SAdd("tags", "golang", "cache", "distributed")
//	members := cache.SMembers("tags")
//	fmt.Printf("Tags: %v\n", members)
//
// Parameters:
//   - key: The set key
//
// Returns:
//   - Slice containing all set members
func (c *Cache) SMembers(key string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) || value.Type != TypeSet {
		return []string{}
	}

	set, ok := value.Data.(map[string]bool)
	if !ok {
		return []string{}
	}
	members := make([]string, 0, len(set))
	for member := range set {
		members = append(members, member)
	}
	return members
}

// SIsMember checks if a member exists in a set.
// Returns true if the member exists in the set, false otherwise.
//
// Example:
//
//	cache.SAdd("tags", "golang", "cache")
//	if cache.SIsMember("tags", "golang") {
//		fmt.Println("golang is in the tags set")
//	}
//
// Parameters:
//   - key: The set key
//   - member: The member to check for
//
// Returns:
//   - Boolean indicating if the member exists in the set
func (c *Cache) SIsMember(key, member string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.data[key]
	if !exists || c.isExpired(value) || value.Type != TypeSet {
		return false
	}

	set, ok := value.Data.(map[string]bool)
	if !ok {
		return false
	}
	return set[member]
}

// Stats returns statistics about the current state of the cache.
// This is useful for monitoring memory usage, key distribution, and expiration status.
//
// Example:
//
//	stats := cache.Stats()
//	fmt.Printf("Total keys: %d\n", stats["keys"])
//	fmt.Printf("Expired keys: %d\n", stats["expired"])
//	if types, ok := stats["types"].(map[string]int); ok {
//		for dataType, count := range types {
//			fmt.Printf("%s keys: %d\n", dataType, count)
//		}
//	}
//
// Returns:
//   - Map containing cache statistics:
//   - "keys": total number of keys
//   - "types": map of data type counts
//   - "expired": number of expired but not yet cleaned up keys
func (c *Cache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"keys": len(c.data),
	}

	typeCount := make(map[string]int)
	expiredCount := 0
	now := time.Now()

	for _, value := range c.data {
		switch value.Type {
		case TypeString:
			typeCount["string"]++
		case TypeHash:
			typeCount["hash"]++
		case TypeList:
			typeCount["list"]++
		case TypeSet:
			typeCount["set"]++
		}

		if !value.ExpiresAt.IsZero() && now.After(value.ExpiresAt) {
			expiredCount++
		}
	}

	stats["types"] = typeCount
	stats["expired"] = expiredCount

	return stats
}
