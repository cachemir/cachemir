// Package cachemir provides the core components for the CacheMir distributed caching system.
//
// This package serves as the main entry point for CacheMir's public API and contains
// the primary interfaces and types used throughout the system. It brings together
// all the individual components to provide a cohesive caching solution.
//
// # Overview
//
// CacheMir is a distributed in-memory caching system designed for high performance
// and horizontal scalability. It provides Redis-compatible operations through a
// lightweight binary protocol and uses client-side consistent hashing for
// automatic key distribution across multiple server nodes.
//
// # Key Features
//
//   - Redis-compatible API with familiar commands
//   - Horizontal scaling through consistent hashing
//   - High-performance binary protocol
//   - Connection pooling and automatic retry logic
//   - Multiple data types: strings, hashes, lists, sets
//   - Automatic expiration and memory management
//   - Thread-safe operations
//   - Production-ready with comprehensive configuration
//
// # Architecture Components
//
// Client SDK (pkg/client):
//   - High-level client library
//   - Automatic node selection via consistent hashing
//   - Connection pooling per server node
//   - Retry logic and error handling
//   - Redis-compatible API
//
// Cache Engine (pkg/cache):
//   - In-memory storage with multiple data types
//   - Automatic expiration cleanup
//   - Thread-safe operations
//   - Memory-efficient storage
//
// Protocol (pkg/protocol):
//   - Lightweight binary protocol
//   - Efficient serialization/deserialization
//   - Command and response framing
//   - Network-optimized data transfer
//
// Consistent Hashing (pkg/hash):
//   - Virtual nodes for better distribution
//   - Minimal key redistribution on topology changes
//   - Thread-safe ring operations
//   - Configurable virtual node count
//
// Configuration (pkg/config):
//   - Server and client configuration management
//   - Command-line flags and environment variables
//   - Validation and defaults
//   - Production-ready settings
//
// Server (internal/server):
//   - TCP server with concurrent connection handling
//   - Command parsing and execution
//   - Integration with cache engine
//   - Graceful shutdown support
//
// # Usage Examples
//
// Basic client usage:
//
//	import "github.com/cachemir/cachemir/pkg/client"
//
//	// Connect to cluster
//	client := client.New([]string{"server1:8080", "server2:8080"})
//	defer client.Close()
//
//	// String operations
//	err := client.Set("user:123", "john_doe", time.Hour)
//	value, err := client.Get("user:123")
//	deleted, err := client.Del("user:123")
//
// Advanced client configuration:
//
//	import "github.com/cachemir/cachemir/pkg/config"
//
//	config := &config.ClientConfig{
//		Nodes:           []string{"node1:8080", "node2:8080", "node3:8080"},
//		MaxConnsPerNode: 50,
//		ConnTimeout:     10,
//		RetryAttempts:   5,
//		VirtualNodes:    300,
//	}
//	client := client.NewWithConfig(config)
//
// Server setup:
//
//	import "github.com/cachemir/cachemir/internal/server"
//	import "github.com/cachemir/cachemir/pkg/config"
//
//	config := config.LoadServerConfig()
//	srv := server.New(config.Port)
//	log.Fatal(srv.Start())
//
// # Data Types and Operations
//
// Strings:
//   - GET, SET, DEL, EXISTS
//   - INCR, DECR for atomic counters
//   - EXPIRE, TTL, PERSIST for expiration
//
// Hashes:
//   - HGET, HSET, HDEL for field operations
//   - HGETALL for retrieving all fields
//   - Perfect for storing objects/records
//
// Lists:
//   - LPUSH, RPUSH for adding elements
//   - LPOP, RPOP for removing elements
//   - LLEN for getting list length
//   - Useful for queues and stacks
//
// Sets:
//   - SADD, SREM for membership operations
//   - SMEMBERS for getting all members
//   - SISMEMBER for membership testing
//   - Great for tags and unique collections
//
// # Scaling and Performance
//
// Horizontal Scaling:
//   - Client-side sharding using consistent hashing
//   - Add nodes without data migration
//   - Linear performance scaling
//   - No single point of failure
//
// Performance Characteristics:
//   - Single node: ~100K operations/second
//   - Sub-millisecond latency on local networks
//   - Memory overhead: ~100 bytes per key
//   - Efficient connection pooling
//
// # Production Considerations
//
// Deployment:
//   - Deploy 3-5 nodes across availability zones
//   - Use load balancers for client discovery
//   - Monitor key distribution and node health
//   - Set appropriate resource limits
//
// Configuration:
//   - Tune connection pool sizes based on load
//   - Configure appropriate timeouts
//   - Set up monitoring and alerting
//   - Use environment variables for configuration
//
// Monitoring:
//   - Track request latency and error rates
//   - Monitor memory usage and key distribution
//   - Set up health checks and alerts
//   - Use PING command for connectivity tests
//
// # Error Handling
//
// The client SDK provides comprehensive error handling:
//   - Network errors trigger automatic retries
//   - Connection failures cause failover to other nodes
//   - Timeout errors are clearly distinguished
//   - Server errors are propagated with context
//
// # Thread Safety
//
// All CacheMir components are designed for concurrent use:
//   - Client SDK is fully thread-safe
//   - Cache engine uses read-write locks
//   - Connection pools handle concurrent access
//   - Consistent hash ring supports concurrent operations
//
// # Integration
//
// CacheMir integrates well with existing systems:
//   - Redis-compatible API for easy migration
//   - Standard Go interfaces and patterns
//   - Configurable through environment variables
//   - Docker and Kubernetes ready
//   - Comprehensive logging and metrics
//
// For detailed documentation of specific components, refer to their individual
// package documentation.
package cachemir
