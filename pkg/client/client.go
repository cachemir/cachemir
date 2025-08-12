// Package client provides a high-level client SDK for connecting to CacheMir cache servers.
//
// The client implements automatic node selection using consistent hashing, connection pooling
// for efficient resource usage, and retry logic for handling transient failures. It provides
// a Redis-compatible API that abstracts away the complexity of distributed caching.
//
// Key Features:
//   - Consistent hashing for automatic node selection
//   - Connection pooling per server node
//   - Automatic retry logic with configurable attempts
//   - Redis-compatible command API
//   - Thread-safe operations
//   - Graceful error handling and failover
//
// Basic Usage:
//
//	// Connect to a cluster
//	client := client.New([]string{"server1:8080", "server2:8080", "server3:8080"})
//	defer client.Close()
//
//	// String operations
//	err := client.Set("user:123", "john_doe", time.Hour)
//	value, err := client.Get("user:123")
//
//	// Hash operations
//	client.HSet("user:123:profile", "name", "John Doe")
//	client.HSet("user:123:profile", "email", "john@example.com")
//	profile, err := client.HGetAll("user:123:profile")
//
//	// List operations
//	length, err := client.LPush("tasks", "task1", "task2", "task3")
//	task, err := client.LPop("tasks")
//
//	// Set operations
//	added, err := client.SAdd("tags", "golang", "cache", "distributed")
//	members, err := client.SMembers("tags")
//
// Advanced Configuration:
//
//	config := &config.ClientConfig{
//		Nodes:           []string{"node1:8080", "node2:8080"},
//		MaxConnsPerNode: 20,
//		ConnTimeout:     10,
//		RetryAttempts:   5,
//		VirtualNodes:    200,
//	}
//	client := client.NewWithConfig(config)
//
// The client automatically handles:
//   - Node selection based on key hashing
//   - Connection establishment and reuse
//   - Network error recovery
//   - Load balancing across nodes
//   - Resource cleanup
package client

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/cachemir/cachemir/pkg/config"
	"github.com/cachemir/cachemir/pkg/hash"
	"github.com/cachemir/cachemir/pkg/protocol"
)

// Client provides a high-level interface to a CacheMir cluster.
// It manages connections to multiple server nodes, automatically selects
// the appropriate node for each key using consistent hashing, and provides
// Redis-compatible operations with built-in retry logic.
//
// The client is thread-safe and can be used concurrently from multiple goroutines.
// It maintains connection pools to each server node for efficient resource usage.
//
// Example:
//
//	client := client.New([]string{"server1:8080", "server2:8080"})
//	defer client.Close()
//
//	// The client automatically selects the right node for each key
//	client.Set("user:123", "data", 0)    // May go to server1
//	client.Set("session:abc", "data", 0) // May go to server2
type Client struct {
	config *config.ClientConfig       // Client configuration
	ring   *hash.ConsistentHash       // Consistent hash ring for node selection
	pools  map[string]*ConnectionPool // Connection pools per node
	mu     sync.RWMutex               // Protects the pools map
}

// ConnectionPool manages a pool of connections to a single server node.
// It provides connection reuse, limits the number of concurrent connections,
// and handles connection lifecycle management.
//
// The pool creates connections on-demand up to the configured maximum,
// and reuses existing connections when available. Connections are returned
// to the pool after use for efficient resource utilization.
type ConnectionPool struct {
	connections chan net.Conn // Pool of available connections
	address     string        // Server address (host:port)
	connTimeout time.Duration // Timeout for creating new connections
	mu          sync.Mutex    // Protects the created counter
	maxConns    int           // Maximum number of connections
	created     int           // Number of connections created
}

// New creates a new Client connected to the specified server nodes.
// It uses default configuration values and creates connection pools for each node.
// The nodes are added to a consistent hash ring for automatic key distribution.
//
// Example:
//
//	// Single node
//	client := client.New([]string{"localhost:8080"})
//
//	// Multi-node cluster
//	client := client.New([]string{
//		"cache1.example.com:8080",
//		"cache2.example.com:8080",
//		"cache3.example.com:8080",
//	})
//	defer client.Close()
//
// Parameters:
//   - nodes: List of server addresses in "host:port" format
//
// Returns:
//   - A new Client instance ready for use
func New(nodes []string) *Client {
	cfg := config.LoadClientConfig()
	cfg.Nodes = nodes

	return NewWithConfig(cfg)
}

// NewWithConfig creates a new Client using the provided configuration.
// This allows fine-tuning of connection pooling, timeouts, retry behavior,
// and consistent hashing parameters.
//
// Example:
//
//	config := &config.ClientConfig{
//		Nodes:           []string{"server1:8080", "server2:8080"},
//		MaxConnsPerNode: 50,        // More connections per node
//		ConnTimeout:     10,        // Longer connection timeout
//		RetryAttempts:   5,         // More retry attempts
//		VirtualNodes:    300,       // Better key distribution
//	}
//	client := client.NewWithConfig(config)
//
// Parameters:
//   - config: Client configuration with all settings
//
// Returns:
//   - A new Client instance configured according to the provided settings
//
// Panics:
//   - If the configuration is invalid (fails validation)
func NewWithConfig(cfg *config.ClientConfig) *Client {
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("invalid client config: %v", err))
	}

	client := &Client{
		config: cfg,
		ring:   hash.New(cfg.VirtualNodes),
		pools:  make(map[string]*ConnectionPool),
	}

	for _, node := range cfg.Nodes {
		client.ring.AddNode(node)
		client.pools[node] = &ConnectionPool{
			address:     node,
			connections: make(chan net.Conn, cfg.MaxConnsPerNode),
			maxConns:    cfg.MaxConnsPerNode,
			connTimeout: time.Duration(cfg.ConnTimeout) * time.Second,
		}
	}

	return client
}

// AddNode dynamically adds a new server node to the cluster.
// The node is added to the consistent hash ring and a connection pool is created.
// Existing keys may be redistributed to the new node according to consistent hashing.
//
// This operation is thread-safe and can be called while the client is in use.
// It's useful for scaling up the cluster or replacing failed nodes.
//
// Example:
//
//	client := client.New([]string{"server1:8080", "server2:8080"})
//
//	// Add a new node to handle more load
//	client.AddNode("server3:8080")
//
//	// Keys will now be distributed across all three nodes
//
// Parameters:
//   - address: Server address in "host:port" format
func (c *Client) AddNode(address string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ring.AddNode(address)
	if _, exists := c.pools[address]; !exists {
		c.pools[address] = &ConnectionPool{
			address:     address,
			connections: make(chan net.Conn, c.config.MaxConnsPerNode),
			maxConns:    c.config.MaxConnsPerNode,
			connTimeout: time.Duration(c.config.ConnTimeout) * time.Second,
		}
	}
}

// RemoveNode dynamically removes a server node from the cluster.
// The node is removed from the consistent hash ring and its connection pool is closed.
// Keys previously assigned to this node will be redistributed to remaining nodes.
//
// This operation is thread-safe and can be called while the client is in use.
// It's useful for handling node failures or scaling down the cluster.
//
// Example:
//
//	client := client.New([]string{"server1:8080", "server2:8080", "server3:8080"})
//
//	// Remove a failed node
//	client.RemoveNode("server2:8080")
//
//	// Keys will be redistributed to remaining nodes
//
// Parameters:
//   - address: Server address to remove
func (c *Client) RemoveNode(address string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ring.RemoveNode(address)
	if pool, exists := c.pools[address]; exists {
		pool.Close()
		delete(c.pools, address)
	}
}

// getConnection obtains a connection to the server responsible for the given key.
// It uses consistent hashing to determine the target node, then gets a connection
// from that node's connection pool.
//
// Returns an error if no nodes are available or if connection establishment fails.
func (c *Client) getConnection(key string) (net.Conn, error) {
	node := c.ring.GetNode(key)
	if node == "" {
		return nil, fmt.Errorf("no available nodes")
	}

	c.mu.RLock()
	pool, exists := c.pools[node]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no connection pool for node: %s", node)
	}

	return pool.Get()
}

// returnConnection returns a connection to the appropriate connection pool.
// The connection is determined by the key's assigned node in the hash ring.
// If the node is no longer available, the connection is closed.
func (c *Client) returnConnection(key string, conn net.Conn) {
	node := c.ring.GetNode(key)
	if node == "" {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
		return
	}

	c.mu.RLock()
	pool, exists := c.pools[node]
	c.mu.RUnlock()

	if exists {
		pool.Put(conn)
	} else {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}
}

// executeCommand executes a command against the appropriate server node with retry logic.
// It automatically selects the correct node based on the command's key, handles
// network errors with retries, and manages connection lifecycle.
//
// The method implements the following retry strategy:
//  1. Determine target node using consistent hashing
//  2. Get connection from node's connection pool
//  3. Send command and read response
//  4. Return connection to pool on success
//  5. Close connection and retry on failure
//  6. Return error after exhausting retry attempts
func (c *Client) executeCommand(cmd *protocol.Command) (*protocol.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		conn, err := c.getConnection(cmd.Key)
		if err != nil {
			lastErr = err
			continue
		}

		writeDeadline := time.Now().Add(time.Duration(c.config.WriteTimeout) * time.Second)
		if writeErr := conn.SetWriteDeadline(writeDeadline); writeErr != nil {
			c.returnConnection(cmd.Key, conn)
			lastErr = writeErr
			continue
		}
		if writeErr := protocol.WriteCommand(conn, cmd); writeErr != nil {
			if closeErr := conn.Close(); closeErr != nil {
				log.Printf("Error closing connection: %v", closeErr)
			}
			lastErr = writeErr
			continue
		}

		readDeadline := time.Now().Add(time.Duration(c.config.ReadTimeout) * time.Second)
		if readErr := conn.SetReadDeadline(readDeadline); readErr != nil {
			c.returnConnection(cmd.Key, conn)
			lastErr = readErr
			continue
		}
		resp, err := protocol.ReadResponse(conn)
		if err != nil {
			if closeErr := conn.Close(); closeErr != nil {
				log.Printf("Error closing connection: %v", closeErr)
			}
			lastErr = err
			continue
		}

		c.returnConnection(cmd.Key, conn)
		return resp, nil
	}

	return nil, fmt.Errorf("command failed after %d attempts: %v", c.config.RetryAttempts+1, lastErr)
}

// Get retrieves the string value of a key.
// Returns an error if the key doesn't exist, has expired, or is not a string value.
//
// Example:
//
//	client.Set("greeting", "Hello, World!", 0)
//	value, err := client.Get("greeting")
//	if err != nil {
//		log.Printf("Key not found: %v", err)
//	} else {
//		fmt.Printf("Greeting: %s\n", value)
//	}
//
// Parameters:
//   - key: The key to retrieve
//
// Returns:
//   - The string value if found
//   - Error if key doesn't exist or operation fails
func (c *Client) Get(key string) (string, error) {
	return c.executeStringCommand(protocol.CmdGet, key, "key not found")
}

// Set stores a string value with an optional expiration time.
// If ttl is 0, the key will not expire. If ttl is positive, the key will
// automatically expire after the specified duration.
//
// Example:
//
//	// Set without expiration
//	err := client.Set("permanent_key", "permanent_value", 0)
//
//	// Set with 1 hour expiration
//	err = client.Set("session_key", "session_data", time.Hour)
//
//	// Set with 30 minutes expiration
//	err = client.Set("cache_key", "cached_data", 30*time.Minute)
//
// Parameters:
//   - key: The key to store
//   - value: The string value to store
//   - ttl: Time-to-live duration (0 for no expiration)
//
// Returns:
//   - Error if the operation fails
func (c *Client) Set(key, value string, ttl time.Duration) error {
	cmd := &protocol.Command{
		Type: protocol.CmdSet,
		Key:  key,
		Args: []string{value},
		TTL:  ttl,
	}

	resp, err := c.executeCommand(cmd)
	if err != nil {
		return err
	}

	if resp.Type == protocol.RespError {
		return fmt.Errorf("server error: %s", resp.Error)
	}

	return nil
}

// Del deletes a key from the cache.
// Returns true if the key existed and was deleted, false if it didn't exist.
//
// Example:
//
//	client.Set("temp_key", "temp_value", 0)
//	deleted, err := client.Del("temp_key")
//	if err != nil {
//		log.Printf("Delete failed: %v", err)
//	} else if deleted {
//		fmt.Println("Key deleted successfully")
//	} else {
//		fmt.Println("Key didn't exist")
//	}
//
// Parameters:
//   - key: The key to delete
//
// Returns:
//   - Boolean indicating if the key was deleted
//   - Error if the operation fails

// executeBoolCommand executes a command that returns a boolean result based on int64 response
func (c *Client) executeBoolCommand(cmdType protocol.CommandType, key string) (bool, error) {
	cmd := &protocol.Command{
		Type: cmdType,
		Key:  key,
	}

	resp, err := c.executeCommand(cmd)
	if err != nil {
		return false, err
	}

	if resp.Type == protocol.RespError {
		return false, fmt.Errorf("server error: %s", resp.Error)
	}

	if resp.Type != protocol.RespInt {
		return false, fmt.Errorf("unexpected response type")
	}

	if val, ok := resp.Data.(int64); ok {
		return val == 1, nil
	}
	return false, fmt.Errorf("response data is not an int64")
}

// executeStringCommand executes a command that returns a string result
func (c *Client) executeStringCommand(cmdType protocol.CommandType, key, nilErrorMsg string) (string, error) {
	cmd := &protocol.Command{
		Type: cmdType,
		Key:  key,
	}

	resp, err := c.executeCommand(cmd)
	if err != nil {
		return "", err
	}

	if resp.Type == protocol.RespNil {
		return "", fmt.Errorf("%s", nilErrorMsg)
	}

	if resp.Type == protocol.RespError {
		return "", fmt.Errorf("server error: %s", resp.Error)
	}

	if resp.Type != protocol.RespString {
		return "", fmt.Errorf("unexpected response type")
	}

	if str, ok := resp.Data.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("response data is not a string")
}

// executeInt64Command executes a command that returns an int64 result
func (c *Client) executeInt64Command(cmdType protocol.CommandType, key string) (int64, error) {
	cmd := &protocol.Command{
		Type: cmdType,
		Key:  key,
	}

	resp, err := c.executeCommand(cmd)
	if err != nil {
		return 0, err
	}

	if resp.Type == protocol.RespError {
		return 0, fmt.Errorf("server error: %s", resp.Error)
	}

	if resp.Type != protocol.RespInt {
		return 0, fmt.Errorf("unexpected response type")
	}

	if val, ok := resp.Data.(int64); ok {
		return val, nil
	}
	return 0, fmt.Errorf("response data is not an int64")
}

// executeInt64CommandWithArgs executes a command with arguments that returns an int64 result
func (c *Client) executeInt64CommandWithArgs(cmdType protocol.CommandType, key string, args []string) (int64, error) {
	cmd := &protocol.Command{
		Type: cmdType,
		Key:  key,
		Args: args,
	}

	resp, err := c.executeCommand(cmd)
	if err != nil {
		return 0, err
	}

	if resp.Type == protocol.RespError {
		return 0, fmt.Errorf("server error: %s", resp.Error)
	}

	if resp.Type != protocol.RespInt {
		return 0, fmt.Errorf("unexpected response type")
	}

	if val, ok := resp.Data.(int64); ok {
		return val, nil
	}
	return 0, fmt.Errorf("response data is not an int64")
}

func (c *Client) Del(key string) (bool, error) {
	return c.executeBoolCommand(protocol.CmdDel, key)
}

// Exists checks if a key exists in the cache.
// Returns true if the key exists and hasn't expired, false otherwise.
//
// Example:
//
//	client.Set("check_key", "value", 0)
//	exists, err := client.Exists("check_key")
//	if err != nil {
//		log.Printf("Check failed: %v", err)
//	} else if exists {
//		fmt.Println("Key exists")
//	} else {
//		fmt.Println("Key doesn't exist")
//	}
//
// Parameters:
//   - key: The key to check
//
// Returns:
//   - Boolean indicating if the key exists
//   - Error if the operation fails
func (c *Client) Exists(key string) (bool, error) {
	return c.executeBoolCommand(protocol.CmdExists, key)
}

// Incr increments the integer value of a key by 1.
// If the key doesn't exist, it's set to 1. If the key exists but contains
// a non-integer value, an error is returned.
//
// Example:
//
//	// First increment creates the key with value 1
//	count, err := client.Incr("page_views")
//	fmt.Printf("Page views: %d\n", count) // Prints: Page views: 1
//
//	// Subsequent increments increase the value
//	count, err = client.Incr("page_views")
//	fmt.Printf("Page views: %d\n", count) // Prints: Page views: 2
//
// Parameters:
//   - key: The key to increment
//
// Returns:
//   - The new integer value after incrementing
//   - Error if the key contains a non-integer value or operation fails
func (c *Client) Incr(key string) (int64, error) {
	return c.executeInt64Command(protocol.CmdIncr, key)
}

// Decr decrements the integer value of a key by 1.
// If the key doesn't exist, it's set to -1. If the key exists but contains
// a non-integer value, an error is returned.
//
// Example:
//
//	client.Set("countdown", "10", 0)
//	count, err := client.Decr("countdown")
//	fmt.Printf("Countdown: %d\n", count) // Prints: Countdown: 9
//
// Parameters:
//   - key: The key to decrement
//
// Returns:
//   - The new integer value after decrementing
//   - Error if the key contains a non-integer value or operation fails
func (c *Client) Decr(key string) (int64, error) {
	return c.executeInt64Command(protocol.CmdDecr, key)
}

// Expire sets a timeout on a key. After the timeout, the key will be automatically deleted.
// Returns true if the timeout was set, false if the key doesn't exist.
//
// Example:
//
//	client.Set("temp_data", "some_value", 0) // No expiration initially
//	success, err := client.Expire("temp_data", 30*time.Second)
//	if success {
//		fmt.Println("Key will expire in 30 seconds")
//	}
//
// Parameters:
//   - key: The key to set expiration on
//   - ttl: Time-to-live duration
//
// Returns:
//   - Boolean indicating if the expiration was set
//   - Error if the operation fails
func (c *Client) Expire(key string, ttl time.Duration) (bool, error) {
	cmd := &protocol.Command{
		Type: protocol.CmdExpire,
		Key:  key,
		TTL:  ttl,
	}

	resp, err := c.executeCommand(cmd)
	if err != nil {
		return false, err
	}

	if resp.Type == protocol.RespError {
		return false, fmt.Errorf("server error: %s", resp.Error)
	}

	if resp.Type != protocol.RespInt {
		return false, fmt.Errorf("unexpected response type")
	}

	if val, ok := resp.Data.(int64); ok {
		return val == 1, nil
	}
	return false, fmt.Errorf("response data is not an int64")
}

// TTL returns the remaining time to live of a key.
// Returns the duration until expiration, or special values:
//   - Negative duration < -1 second: key doesn't exist
//   - -1 second: key exists but has no expiration
//   - 0 or positive: remaining time until expiration
//
// Example:
//
//	client.Set("temp_key", "value", time.Minute)
//	ttl, err := client.TTL("temp_key")
//	if err != nil {
//		log.Printf("TTL check failed: %v", err)
//	} else if ttl > 0 {
//		fmt.Printf("Key expires in %v\n", ttl)
//	} else if ttl == -1*time.Second {
//		fmt.Println("Key has no expiration")
//	} else {
//		fmt.Println("Key doesn't exist")
//	}
//
// Parameters:
//   - key: The key to check
//
// Returns:
//   - Remaining time to live, or special negative values
//   - Error if the operation fails
func (c *Client) TTL(key string) (time.Duration, error) {
	cmd := &protocol.Command{
		Type: protocol.CmdTTL,
		Key:  key,
	}

	resp, err := c.executeCommand(cmd)
	if err != nil {
		return 0, err
	}

	if resp.Type == protocol.RespError {
		return 0, fmt.Errorf("server error: %s", resp.Error)
	}

	if resp.Type != protocol.RespInt {
		return 0, fmt.Errorf("unexpected response type")
	}

	seconds, ok := resp.Data.(int64)
	if !ok {
		return 0, fmt.Errorf("response data is not an int64")
	}
	return time.Duration(seconds) * time.Second, nil
}

// HGet retrieves the value of a hash field.
// Returns an error if the hash doesn't exist, has expired, or the field doesn't exist.
//
// Example:
//
//	client.HSet("user:123", "name", "John Doe")
//	client.HSet("user:123", "email", "john@example.com")
//
//	name, err := client.HGet("user:123", "name")
//	if err != nil {
//		log.Printf("Field not found: %v", err)
//	} else {
//		fmt.Printf("User name: %s\n", name)
//	}
//
// Parameters:
//   - key: The hash key
//   - field: The field name within the hash
//
// Returns:
//   - The field value if found
//   - Error if hash or field doesn't exist, or operation fails
func (c *Client) HGet(key, field string) (string, error) {
	cmd := &protocol.Command{
		Type: protocol.CmdHGet,
		Key:  key,
		Args: []string{field},
	}

	resp, err := c.executeCommand(cmd)
	if err != nil {
		return "", err
	}

	if resp.Type == protocol.RespNil {
		return "", fmt.Errorf("field not found")
	}

	if resp.Type == protocol.RespError {
		return "", fmt.Errorf("server error: %s", resp.Error)
	}

	if resp.Type != protocol.RespString {
		return "", fmt.Errorf("unexpected response type")
	}

	if str, ok := resp.Data.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("response data is not a string")
}

// HSet sets the value of a hash field.
// If the hash doesn't exist, it's created. If the field exists, its value is updated.
//
// Example:
//
//	// Create a user profile hash
//	client.HSet("user:123", "name", "John Doe")
//	client.HSet("user:123", "email", "john@example.com")
//	client.HSet("user:123", "age", "30")
//	client.HSet("user:123", "city", "New York")
//
// Parameters:
//   - key: The hash key
//   - field: The field name within the hash
//   - value: The field value to set
//
// Returns:
//   - Error if the operation fails
func (c *Client) HSet(key, field, value string) error {
	cmd := &protocol.Command{
		Type: protocol.CmdHSet,
		Key:  key,
		Args: []string{field, value},
	}

	resp, err := c.executeCommand(cmd)
	if err != nil {
		return err
	}

	if resp.Type == protocol.RespError {
		return fmt.Errorf("server error: %s", resp.Error)
	}

	return nil
}

// HGetAll retrieves all fields and values in a hash.
// Returns a map of field-value pairs. If the hash doesn't exist or has expired,
// returns an empty map.
//
// Example:
//
//	client.HSet("user:123", "name", "John Doe")
//	client.HSet("user:123", "email", "john@example.com")
//	client.HSet("user:123", "age", "30")
//
//	profile, err := client.HGetAll("user:123")
//	if err != nil {
//		log.Printf("Failed to get profile: %v", err)
//	} else {
//		for field, value := range profile {
//			fmt.Printf("%s: %s\n", field, value)
//		}
//	}
//
// Parameters:
//   - key: The hash key
//
// Returns:
//   - Map of all field-value pairs in the hash
//   - Error if the operation fails
func (c *Client) HGetAll(key string) (map[string]string, error) {
	cmd := &protocol.Command{
		Type: protocol.CmdHGetAll,
		Key:  key,
	}

	resp, err := c.executeCommand(cmd)
	if err != nil {
		return nil, err
	}

	if resp.Type == protocol.RespError {
		return nil, fmt.Errorf("server error: %s", resp.Error)
	}

	if resp.Type != protocol.RespArray {
		return nil, fmt.Errorf("unexpected response type")
	}

	arr, ok := resp.Data.([]string)
	if !ok {
		return nil, fmt.Errorf("response data is not a string array")
	}
	result := make(map[string]string)

	for i := 0; i < len(arr); i += 2 {
		if i+1 < len(arr) {
			result[arr[i]] = arr[i+1]
		}
	}

	return result, nil
}

// LPush inserts values at the head (left) of a list.
// If the list doesn't exist, it's created. Values are inserted in reverse order,
// so the last value in the arguments becomes the first element in the list.
// Returns the new length of the list after insertion.
//
// Example:
//
//	// Creates list: ["task3", "task2", "task1"]
//	length, err := client.LPush("todo", "task1", "task2", "task3")
//	fmt.Printf("Todo list has %d items\n", length)
//
//	// Add more tasks: ["urgent", "task3", "task2", "task1"]
//	length, err = client.LPush("todo", "urgent")
//
// Parameters:
//   - key: The list key
//   - values: Values to insert at the head
//
// Returns:
//   - The new length of the list after insertion
//   - Error if the operation fails
func (c *Client) LPush(key string, values ...string) (int64, error) {
	return c.executeInt64CommandWithArgs(protocol.CmdLPush, key, values)
}

// RPush inserts values at the tail (right) of a list.
// If the list doesn't exist, it's created. Values are appended in order.
// Returns the new length of the list after insertion.
//
// Example:
//
//	// Creates list: ["item1", "item2", "item3"]
//	length, err := client.RPush("queue", "item1", "item2", "item3")
//	fmt.Printf("Queue has %d items\n", length)
//
//	// Add more items: ["item1", "item2", "item3", "item4"]
//	length, err = client.RPush("queue", "item4")
//
// Parameters:
//   - key: The list key
//   - values: Values to insert at the tail
//
// Returns:
//   - The new length of the list after insertion
//   - Error if the operation fails
func (c *Client) RPush(key string, values ...string) (int64, error) {
	return c.executeInt64CommandWithArgs(protocol.CmdRPush, key, values)
}

// LPop removes and returns the first element from the head (left) of a list.
// Returns an error if the list doesn't exist, has expired, or is empty.
//
// Example:
//
//	client.LPush("tasks", "task1", "task2", "task3")
//	// List is now: ["task3", "task2", "task1"]
//
//	task, err := client.LPop("tasks")
//	if err != nil {
//		log.Printf("No tasks available: %v", err)
//	} else {
//		fmt.Printf("Processing task: %s\n", task) // Prints: task3
//	}
//
// Parameters:
//   - key: The list key
//
// Returns:
//   - The first element if successful
//   - Error if list doesn't exist, is empty, or operation fails
func (c *Client) LPop(key string) (string, error) {
	return c.executeStringCommand(protocol.CmdLPop, key, "list is empty")
}

// SAdd adds members to a set.
// If the set doesn't exist, it's created. Duplicate members are ignored.
// Returns the number of members that were actually added (not counting duplicates).
//
// Example:
//
//	// Add tags to an article
//	added, err := client.SAdd("article:123:tags", "golang", "programming", "tutorial")
//	fmt.Printf("Added %d new tags\n", added) // Prints: Added 3 new tags
//
//	// Add more tags (including a duplicate)
//	added, err = client.SAdd("article:123:tags", "golang", "cache", "performance")
//	fmt.Printf("Added %d new tags\n", added) // Prints: Added 2 new tags
//
// Parameters:
//   - key: The set key
//   - members: Members to add to the set
//
// Returns:
//   - The number of members actually added (excluding duplicates)
//   - Error if the operation fails
func (c *Client) SAdd(key string, members ...string) (int64, error) {
	return c.executeInt64CommandWithArgs(protocol.CmdSAdd, key, members)
}

// SMembers returns all members of a set.
// Returns an empty slice if the set doesn't exist, has expired, or is empty.
// The order of members is not guaranteed.
//
// Example:
//
//	client.SAdd("languages", "go", "python", "javascript", "rust")
//
//	members, err := client.SMembers("languages")
//	if err != nil {
//		log.Printf("Failed to get members: %v", err)
//	} else {
//		fmt.Printf("Supported languages: %v\n", members)
//		// Output might be: [go rust python javascript] (order varies)
//	}
//
// Parameters:
//   - key: The set key
//
// Returns:
//   - Slice containing all set members
//   - Error if the operation fails
func (c *Client) SMembers(key string) ([]string, error) {
	cmd := &protocol.Command{
		Type: protocol.CmdSMembers,
		Key:  key,
	}

	resp, err := c.executeCommand(cmd)
	if err != nil {
		return nil, err
	}

	if resp.Type == protocol.RespError {
		return nil, fmt.Errorf("server error: %s", resp.Error)
	}

	if resp.Type != protocol.RespArray {
		return nil, fmt.Errorf("unexpected response type")
	}

	if arr, ok := resp.Data.([]string); ok {
		return arr, nil
	}
	return nil, fmt.Errorf("response data is not a string array")
}

// Ping tests connectivity to the cluster.
// Returns nil if at least one node is reachable, or an error if all nodes are unreachable.
// This is useful for health checks and connection validation.
//
// Example:
//
//	client := client.New([]string{"server1:8080", "server2:8080"})
//
//	if err := client.Ping(); err != nil {
//		log.Printf("Cluster is unreachable: %v", err)
//	} else {
//		fmt.Println("Connected to CacheMir cluster")
//	}
//
// Returns:
//   - nil if cluster is reachable
//   - Error if no nodes are reachable
func (c *Client) Ping() error {
	cmd := &protocol.Command{
		Type: protocol.CmdPing,
	}

	resp, err := c.executeCommand(cmd)
	if err != nil {
		return err
	}

	if resp.Type == protocol.RespError {
		return fmt.Errorf("server error: %s", resp.Error)
	}

	return nil
}

// Close gracefully shuts down the client by closing all connection pools.
// This should be called when the client is no longer needed to free resources.
// After calling Close(), the client should not be used for further operations.
//
// Example:
//
//	client := client.New([]string{"server1:8080", "server2:8080"})
//	defer client.Close() // Ensure cleanup
//
//	// Use client for operations...
//	client.Set("key", "value", 0)
//
//	// Close is called automatically by defer
//
// Returns:
//   - Error if there was a problem closing connections (usually nil)
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, pool := range c.pools {
		pool.Close()
	}

	return nil
}

// Get obtains a connection from the pool, creating a new one if necessary.
// It implements connection pooling with a maximum limit per node.
// If the pool is full and at capacity, it waits for an available connection.
func (cp *ConnectionPool) Get() (net.Conn, error) {
	select {
	case conn := <-cp.connections:
		return conn, nil
	default:
		cp.mu.Lock()
		if cp.created < cp.maxConns {
			cp.created++
			cp.mu.Unlock()

			dialer := &net.Dialer{Timeout: cp.connTimeout}
			conn, err := dialer.DialContext(context.Background(), "tcp", cp.address)
			if err != nil {
				cp.mu.Lock()
				cp.created--
				cp.mu.Unlock()
				return nil, err
			}
			return conn, nil
		}
		cp.mu.Unlock()

		select {
		case conn := <-cp.connections:
			return conn, nil
		case <-time.After(cp.connTimeout):
			return nil, fmt.Errorf("connection pool timeout")
		}
	}
}

// Put returns a connection to the pool for reuse.
// If the pool is full, the connection is closed instead of being stored.
func (cp *ConnectionPool) Put(conn net.Conn) {
	select {
	case cp.connections <- conn:
	default:
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
		cp.mu.Lock()
		cp.created--
		cp.mu.Unlock()
	}
}

// Close shuts down the connection pool by closing all pooled connections.
// This is called when a node is removed or the client is shut down.
func (cp *ConnectionPool) Close() {
	close(cp.connections)
	for conn := range cp.connections {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}
}
