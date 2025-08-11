# CacheMir Architecture

## Overview

CacheMir is a distributed in-memory caching solution designed for horizontal scalability using client-side consistent hashing. It provides Redis-compatible commands through a lightweight binary protocol.

## Core Components

### 1. Server (`internal/server/`)
- **Purpose**: Handles client connections and executes cache commands
- **Features**:
  - TCP server with concurrent connection handling
  - Command parsing and execution
  - Graceful shutdown support
  - Configurable timeouts and limits

### 2. Cache Engine (`pkg/cache/`)
- **Purpose**: In-memory storage with Redis-like data structures
- **Data Types**:
  - Strings with TTL support
  - Hashes (field-value pairs)
  - Lists (ordered collections)
  - Sets (unique members)
- **Features**:
  - Automatic expiration cleanup
  - Thread-safe operations
  - Memory-only storage

### 3. Protocol (`pkg/protocol/`)
- **Purpose**: Lightweight binary protocol for client-server communication
- **Features**:
  - Efficient serialization/deserialization
  - Command and response framing
  - Error handling
  - Backward compatibility support

### 4. Consistent Hashing (`pkg/hash/`)
- **Purpose**: Distribute keys across multiple nodes
- **Features**:
  - Virtual nodes for better distribution
  - Minimal key redistribution on node changes
  - Configurable virtual node count
  - Thread-safe operations

### 5. Client SDK (`pkg/client/`)
- **Purpose**: Client library with automatic node selection
- **Features**:
  - Connection pooling per node
  - Automatic retry logic
  - Consistent hashing integration
  - Redis-compatible API

## Data Flow

```
Client Request
    ↓
Consistent Hash (key → node)
    ↓
Connection Pool (get connection)
    ↓
Binary Protocol (serialize command)
    ↓
Network (TCP)
    ↓
Server (deserialize & execute)
    ↓
Cache Engine (data operation)
    ↓
Server (serialize response)
    ↓
Network (TCP)
    ↓
Client (deserialize response)
```

## Scaling Strategy

### Horizontal Scaling
- **Client-side sharding**: Each client maintains the full node list
- **No inter-node communication**: Nodes operate independently
- **Consistent hashing**: Minimizes key redistribution when nodes are added/removed
- **Connection pooling**: Efficient resource utilization

### Adding Nodes
1. Start new CacheMir server instance
2. Update client configuration with new node
3. Consistent hashing automatically redistributes keys
4. No data migration required (cache warming happens naturally)

### Removing Nodes
1. Remove node from client configuration
2. Consistent hashing redirects keys to remaining nodes
3. Stop the removed server instance
4. Cache misses will populate data on new nodes

## Performance Characteristics

### Throughput
- **Single node**: ~100K ops/sec (depending on hardware)
- **Cluster**: Linear scaling with node count
- **Bottlenecks**: Network I/O, client connection limits

### Latency
- **Local network**: <1ms average
- **Cross-datacenter**: Network latency + ~0.1ms processing
- **Connection pooling**: Eliminates connection setup overhead

### Memory Usage
- **Per key overhead**: ~100 bytes (including metadata)
- **Data structures**: Efficient Go native types
- **Garbage collection**: Optimized for low-latency operations

## Consistency Model

### Eventual Consistency
- **No synchronization**: Between nodes
- **Client responsibility**: Consistent key routing
- **Trade-offs**: Availability and partition tolerance over consistency

### Data Durability
- **Memory-only**: No persistence to disk
- **Restart behavior**: All data lost on server restart
- **Backup strategy**: Application-level data replication if needed

## Security Considerations

### Network Security
- **Plain TCP**: No built-in encryption (use TLS proxy if needed)
- **Authentication**: Not implemented (rely on network security)
- **Authorization**: Not implemented (application-level controls)

### Operational Security
- **Resource limits**: Configurable connection and memory limits
- **DoS protection**: Connection timeouts and rate limiting
- **Monitoring**: Built-in stats and health checks

## Configuration

### Server Configuration
- Port and host binding
- Connection limits and timeouts
- Logging levels
- Resource constraints

### Client Configuration
- Node list and discovery
- Connection pooling parameters
- Retry policies and timeouts
- Consistent hashing parameters

## Monitoring and Observability

### Metrics
- **Server**: Connection count, command rates, memory usage
- **Client**: Request latency, error rates, connection pool stats
- **Cache**: Hit/miss ratios, key distribution, expiration rates

### Health Checks
- **PING command**: Basic connectivity test
- **Stats endpoint**: Detailed operational metrics
- **Connection monitoring**: Pool health and node availability

## Deployment Patterns

### Development
- Single node for local development
- Docker container for consistent environments
- Make targets for common operations

### Production
- Multiple nodes across availability zones
- Load balancer for client discovery
- Monitoring and alerting integration
- Automated scaling based on metrics

### High Availability
- Odd number of nodes (3, 5, 7)
- Geographic distribution
- Client-side failover logic
- Health monitoring and automatic node removal
