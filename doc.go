// Package cachemir provides a distributed in-memory caching solution with Redis-compatible operations.
//
// CacheMir is designed for horizontal scalability using client-side consistent hashing,
// offering high performance through a lightweight binary protocol and connection pooling.
// It supports multiple data types and provides automatic expiration with thread-safe operations.
//
// # Architecture Overview
//
// CacheMir consists of several key components:
//
//   - Server: TCP server handling client connections and cache operations
//   - Client SDK: High-level client library with automatic node selection
//   - Cache Engine: In-memory storage with Redis-like data structures
//   - Protocol: Lightweight binary protocol for efficient communication
//   - Consistent Hashing: Distributes keys across nodes with minimal redistribution
//   - Configuration: Flexible configuration through flags and environment variables
//
// # Quick Start
//
// Server:
//
//	import "github.com/cachemir/cachemir/internal/server"
//	import "github.com/cachemir/cachemir/pkg/config"
//
//	config := config.LoadServerConfig()
//	srv := server.New(config.Port)
//	log.Fatal(srv.Start())
//
// Client:
//
//	import "github.com/cachemir/cachemir/pkg/client"
//
//	client := client.New([]string{"localhost:8080", "localhost:8081"})
//	defer client.Close()
//
//	// String operations
//	client.Set("user:123", "john_doe", time.Hour)
//	value, err := client.Get("user:123")
//
//	// Hash operations
//	client.HSet("user:123:profile", "name", "John Doe")
//	profile, err := client.HGetAll("user:123:profile")
//
//	// List operations
//	client.LPush("tasks", "task1", "task2", "task3")
//	task, err := client.LPop("tasks")
//
//	// Set operations
//	client.SAdd("tags", "golang", "cache", "distributed")
//	members, err := client.SMembers("tags")
//
// # Supported Operations
//
// String Operations:
//   - GET, SET, DEL, EXISTS: Basic key-value operations
//   - INCR, DECR: Atomic integer operations
//   - EXPIRE, TTL, PERSIST: Expiration management
//
// Hash Operations:
//   - HGET, HSET, HDEL: Field-level operations
//   - HGETALL: Retrieve all fields and values
//
// List Operations:
//   - LPUSH, RPUSH: Add elements to head/tail
//   - LPOP, RPOP: Remove elements from head/tail
//   - LLEN: Get list length
//
// Set Operations:
//   - SADD, SREM: Add/remove set members
//   - SMEMBERS: Get all set members
//   - SISMEMBER: Check membership
//
// # Scaling and Distribution
//
// CacheMir uses client-side consistent hashing for horizontal scaling:
//
//   - Keys are automatically distributed across multiple server nodes
//   - Adding/removing nodes causes minimal key redistribution
//   - No inter-node communication required
//   - Linear scaling with node count
//
// # Configuration
//
// Server configuration via flags or environment variables:
//
//	./cachemir-server -port 8080 -max-conns 1000
//	# or
//	CACHEMIR_PORT=8080 CACHEMIR_MAX_CONNS=1000 ./cachemir-server
//
// Client configuration:
//
//	config := &config.ClientConfig{
//		Nodes:           []string{"server1:8080", "server2:8080"},
//		MaxConnsPerNode: 20,
//		RetryAttempts:   3,
//		VirtualNodes:    150,
//	}
//	client := client.NewWithConfig(config)
//
// # Performance Characteristics
//
//   - Single node: ~100K ops/sec (hardware dependent)
//   - Cluster: Linear scaling with node count
//   - Latency: <1ms on local network
//   - Memory: ~100 bytes overhead per key
//   - Connection pooling: Efficient resource utilization
//
// # Production Deployment
//
// Recommended setup:
//   - 3-5 nodes across availability zones
//   - Load balancer for client discovery
//   - Monitoring and alerting
//   - Resource limits and health checks
//
// Docker deployment:
//
//	docker run -p 8080:8080 cachemir:latest
//
// # Package Structure
//
//   - pkg/client: Client SDK with consistent hashing
//   - pkg/cache: In-memory cache engine
//   - pkg/protocol: Binary communication protocol
//   - pkg/hash: Consistent hashing implementation
//   - pkg/config: Configuration management
//   - internal/server: Server implementation
//   - cmd/server: Server executable
//   - cmd/client-example: Example client usage
//
// For detailed documentation of individual packages, see their respective godoc pages.
package main
