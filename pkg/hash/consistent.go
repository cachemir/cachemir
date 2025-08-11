// Package hash provides consistent hashing implementation for distributed caching.
//
// Consistent hashing is a technique used to distribute keys across multiple nodes
// in a way that minimizes redistribution when nodes are added or removed from the cluster.
// This implementation uses virtual nodes to achieve better key distribution.
//
// Example usage:
//
//	ch := hash.New(150) // 150 virtual nodes per physical node
//	ch.AddNode("server1:8080")
//	ch.AddNode("server2:8080")
//	ch.AddNode("server3:8080")
//
//	// Get the node responsible for a key
//	node := ch.GetNode("user:123")
//	fmt.Printf("Key 'user:123' maps to node: %s\n", node)
//
// The consistent hash ring ensures that:
//   - Keys are distributed roughly evenly across nodes
//   - Adding/removing nodes only affects a small portion of keys
//   - The same key always maps to the same node (until topology changes)
package hash

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"sync"
)

// DefaultVirtualNodes is the default number of virtual nodes per physical node.
// Virtual nodes help achieve better key distribution across the hash ring.
// A higher number provides better distribution but uses more memory.
const DefaultVirtualNodes = 150

// ConsistentHash implements a consistent hashing ring with virtual nodes.
// It provides thread-safe operations for adding/removing nodes and
// mapping keys to nodes in a distributed system.
//
// The hash ring uses SHA-256 for hashing and maintains virtual nodes
// to ensure better key distribution. When nodes are added or removed,
// only a fraction of keys need to be redistributed.
type ConsistentHash struct {
	mu           sync.RWMutex      // Protects all fields
	ring         map[uint32]string // Hash -> node mapping
	sortedHashes []uint32          // Sorted hash values for binary search
	nodes        map[string]bool   // Set of active nodes
	virtualNodes int               // Number of virtual nodes per physical node
}

// New creates a new ConsistentHash with the specified number of virtual nodes.
// If virtualNodes is <= 0, DefaultVirtualNodes is used.
//
// Virtual nodes are replicas of each physical node placed at different
// positions on the hash ring. More virtual nodes provide better distribution
// but consume more memory.
//
// Example:
//
//	ch := hash.New(100) // 100 virtual nodes per physical node
func New(virtualNodes int) *ConsistentHash {
	if virtualNodes <= 0 {
		virtualNodes = DefaultVirtualNodes
	}
	return &ConsistentHash{
		ring:         make(map[uint32]string),
		nodes:        make(map[string]bool),
		virtualNodes: virtualNodes,
	}
}

// AddNode adds a physical node to the consistent hash ring.
// The node will be replicated virtualNodes times around the ring.
// If the node already exists, this operation is a no-op.
//
// Adding a node will cause some keys to be redistributed to the new node,
// but the majority of keys will remain on their current nodes.
//
// Example:
//
//	ch.AddNode("server1:8080")
//	ch.AddNode("server2:8080")
//
// Parameters:
//   - node: The node identifier (typically "host:port")
func (c *ConsistentHash) AddNode(node string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.nodes[node] {
		return
	}

	c.nodes[node] = true
	for i := 0; i < c.virtualNodes; i++ {
		virtualKey := fmt.Sprintf("%s:%d", node, i)
		hash := c.hashKey(virtualKey)
		c.ring[hash] = node
		c.sortedHashes = append(c.sortedHashes, hash)
	}
	sort.Slice(c.sortedHashes, func(i, j int) bool {
		return c.sortedHashes[i] < c.sortedHashes[j]
	})
}

// RemoveNode removes a physical node from the consistent hash ring.
// All virtual nodes for this physical node are removed.
// If the node doesn't exist, this operation is a no-op.
//
// Removing a node will cause keys previously assigned to this node
// to be redistributed to the remaining nodes.
//
// Example:
//
//	ch.RemoveNode("server1:8080")
//
// Parameters:
//   - node: The node identifier to remove
func (c *ConsistentHash) RemoveNode(node string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.nodes[node] {
		return
	}

	delete(c.nodes, node)
	for i := 0; i < c.virtualNodes; i++ {
		virtualKey := fmt.Sprintf("%s:%d", node, i)
		hash := c.hashKey(virtualKey)
		delete(c.ring, hash)
	}

	var newSortedHashes []uint32
	for _, hash := range c.sortedHashes {
		if _, exists := c.ring[hash]; exists {
			newSortedHashes = append(newSortedHashes, hash)
		}
	}
	c.sortedHashes = newSortedHashes
}

// GetNode returns the node responsible for the given key.
// It uses consistent hashing to determine which node should handle the key.
// Returns an empty string if no nodes are available.
//
// The same key will always return the same node unless the ring topology
// changes (nodes added/removed).
//
// Example:
//
//	node := ch.GetNode("user:123")
//	if node != "" {
//		// Send request to this node
//		fmt.Printf("Route key to node: %s\n", node)
//	}
//
// Parameters:
//   - key: The key to hash and locate
//
// Returns:
//   - The node identifier responsible for this key, or empty string if no nodes
func (c *ConsistentHash) GetNode(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.ring) == 0 {
		return ""
	}

	hash := c.hashKey(key)
	idx := c.search(hash)
	return c.ring[c.sortedHashes[idx]]
}

// GetNodes returns a slice of all active nodes in the hash ring.
// The order is not guaranteed.
//
// Example:
//
//	nodes := ch.GetNodes()
//	fmt.Printf("Active nodes: %v\n", nodes)
//
// Returns:
//   - Slice of node identifiers currently in the ring
func (c *ConsistentHash) GetNodes() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes := make([]string, 0, len(c.nodes))
	for node := range c.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// search performs binary search to find the first hash >= the given hash.
// If no such hash exists, it wraps around to the first hash (index 0).
// This implements the circular nature of the hash ring.
func (c *ConsistentHash) search(hash uint32) int {
	idx := sort.Search(len(c.sortedHashes), func(i int) bool {
		return c.sortedHashes[i] >= hash
	})
	if idx == len(c.sortedHashes) {
		idx = 0
	}
	return idx
}

// hashKey computes a 32-bit hash of the given key using SHA-256.
// Only the first 4 bytes of the SHA-256 hash are used to create
// a 32-bit hash value for ring positioning.
func (c *ConsistentHash) hashKey(key string) uint32 {
	h := sha256.Sum256([]byte(key))
	return uint32(h[0])<<24 | uint32(h[1])<<16 | uint32(h[2])<<8 | uint32(h[3])
}

// Stats returns statistics about the current state of the hash ring.
// This is useful for monitoring and debugging the distribution.
//
// Example:
//
//	stats := ch.Stats()
//	fmt.Printf("Nodes: %d, Virtual nodes: %d\n",
//		stats["nodes"], stats["virtual_nodes"])
//
// Returns:
//   - Map containing statistics:
//   - "nodes": number of physical nodes
//   - "virtual_nodes": total number of virtual nodes
//   - "ring_size": size of the sorted hash array
func (c *ConsistentHash) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"nodes":         len(c.nodes),
		"virtual_nodes": len(c.ring),
		"ring_size":     len(c.sortedHashes),
	}
}
