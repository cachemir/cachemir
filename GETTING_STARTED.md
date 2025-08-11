# Getting Started with CacheMir

CacheMir is a high-performance, horizontally scalable in-memory caching solution with Redis-compatible commands and client-side consistent hashing.

## Quick Start

### 1. Build the Project

```bash
# Build server and client binaries
make build

# Or build individually
go build -o bin/cachemir-server cmd/server/main.go
go build -o bin/cachemir-client-example cmd/client-example/main.go
```

### 2. Start a Server

```bash
# Start single server on port 8080
./bin/cachemir-server

# Or with custom port
./bin/cachemir-server -port 8081

# Or using go run
go run cmd/server/main.go -port 8080
```

### 3. Start a Cluster

```bash
# Start 3-node cluster
make run-cluster

# Or manually
go run cmd/server/main.go -port 8080 &
go run cmd/server/main.go -port 8081 &
go run cmd/server/main.go -port 8082 &
```

### 4. Run Client Example

```bash
# Run the example client
./bin/cachemir-client-example

# Or using go run
go run cmd/client-example/main.go
```

## Basic Usage

### Import the Client SDK

```go
import "github.com/cachemir/cachemir/pkg/client"
```

### Connect to Cluster

```go
// Single node
client := client.New([]string{"localhost:8080"})
defer client.Close()

// Multiple nodes (recommended)
client := client.New([]string{
    "localhost:8080",
    "localhost:8081", 
    "localhost:8082",
})
defer client.Close()
```

### Basic Operations

```go
// Set a key-value pair
err := client.Set("user:123", "john_doe", 0)

// Get a value
value, err := client.Get("user:123")

// Delete a key
deleted, err := client.Del("user:123")

// Check if key exists
exists, err := client.Exists("user:123")

// Increment counter
count, err := client.Incr("page_views")

// Set with expiration
err = client.Set("session:abc", "data", 30*time.Minute)
```

### Hash Operations

```go
// Set hash fields
client.HSet("user:123", "name", "John Doe")
client.HSet("user:123", "email", "john@example.com")

// Get hash field
name, err := client.HGet("user:123", "name")

// Get all hash fields
profile, err := client.HGetAll("user:123")
```

### List Operations

```go
// Add to list
length, err := client.LPush("tasks", "task1", "task2")
length, err = client.RPush("tasks", "task3", "task4")

// Remove from list
item, err := client.LPop("tasks")
item, err = client.RPop("tasks")
```

### Set Operations

```go
// Add to set
added, err := client.SAdd("tags", "golang", "cache", "distributed")

// Get all set members
members, err := client.SMembers("tags")
```

## Configuration

### Server Configuration

```bash
# Command line flags
./bin/cachemir-server \
  -port 8080 \
  -host 0.0.0.0 \
  -max-conns 1000 \
  -read-timeout 30 \
  -write-timeout 10

# Environment variables
export CACHEMIR_PORT=8080
export CACHEMIR_HOST=0.0.0.0
export CACHEMIR_MAX_CONNS=1000
```

### Client Configuration

```bash
# Environment variables
export CACHEMIR_NODES="localhost:8080,localhost:8081,localhost:8082"
export CACHEMIR_MAX_CONNS_PER_NODE=20
export CACHEMIR_CONN_TIMEOUT=10
export CACHEMIR_RETRY_ATTEMPTS=5
```

## Development

### Run Tests

```bash
# All tests
make test

# With race detection
make test-race

# Specific package
go test ./pkg/cache -v
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make vet

# All quality checks
make check
```

### Docker

```bash
# Build Docker image
make docker-build

# Run in container
make docker-run

# Or manually
docker build -t cachemir:latest .
docker run -p 8080:8080 cachemir:latest
```

## Production Deployment

### Recommended Setup

1. **Multiple Nodes**: Deploy 3-5 nodes across different availability zones
2. **Load Balancer**: Use a load balancer for client discovery
3. **Monitoring**: Set up metrics collection and alerting
4. **Resource Limits**: Configure appropriate memory and connection limits

### Example Production Config

```yaml
# docker-compose.yml
version: '3.8'
services:
  cachemir-1:
    image: cachemir:latest
    ports:
      - "8080:8080"
    environment:
      - CACHEMIR_PORT=8080
      - CACHEMIR_MAX_CONNS=2000
    
  cachemir-2:
    image: cachemir:latest
    ports:
      - "8081:8080"
    environment:
      - CACHEMIR_PORT=8080
      - CACHEMIR_MAX_CONNS=2000
    
  cachemir-3:
    image: cachemir:latest
    ports:
      - "8082:8080"
    environment:
      - CACHEMIR_PORT=8080
      - CACHEMIR_MAX_CONNS=2000
```

### Client Configuration for Production

```go
config := &config.ClientConfig{
    Nodes: []string{
        "cachemir-1.example.com:8080",
        "cachemir-2.example.com:8080", 
        "cachemir-3.example.com:8080",
    },
    MaxConnsPerNode: 50,
    ConnTimeout:     10,
    ReadTimeout:     30,
    WriteTimeout:    10,
    RetryAttempts:   3,
    VirtualNodes:    200,
}

client := client.NewWithConfig(config)
```

## Monitoring

### Health Checks

```go
// Basic connectivity test
err := client.Ping()
if err != nil {
    log.Printf("Cluster health check failed: %v", err)
}
```

### Metrics

Monitor these key metrics:

- **Client-side**: Request latency, error rates, connection pool utilization
- **Server-side**: Connection count, command rates, memory usage
- **System**: CPU, memory, network I/O

## Troubleshooting

### Common Issues

1. **Connection Refused**: Check if server is running and port is correct
2. **Timeout Errors**: Increase timeout values or check network latency
3. **Key Not Found**: Verify key exists and hasn't expired
4. **Poor Performance**: Check connection pool settings and network latency

### Debug Mode

```bash
# Enable debug logging
./bin/cachemir-server -log-level debug
```

### Connection Testing

```bash
# Test connectivity with telnet
telnet localhost 8080

# Or use the client example
go run cmd/client-example/main.go
```

## Next Steps

- Read the [Architecture Documentation](docs/ARCHITECTURE.md)
- Check the [API Reference](docs/API.md)
- Explore the [Examples](examples/)
- Run the comprehensive [Tests](tests/)

## Support

- **Issues**: Report bugs and feature requests on GitHub
- **Documentation**: Check the `docs/` directory
- **Examples**: See `examples/` and `cmd/client-example/`
