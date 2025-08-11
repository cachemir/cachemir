# CacheMir API Reference

## Client SDK Usage

### Initialization

```go
import "github.com/cachemir/cachemir/pkg/client"

// Simple initialization
client := client.New([]string{"localhost:8080", "localhost:8081"})
defer client.Close()

// With custom configuration
config := &config.ClientConfig{
    Nodes:           []string{"node1:8080", "node2:8080"},
    MaxConnsPerNode: 20,
    ConnTimeout:     10,
    RetryAttempts:   5,
    VirtualNodes:    200,
}
client := client.NewWithConfig(config)
```

## String Operations

### GET
Retrieve the value of a key.

```go
value, err := client.Get("mykey")
if err != nil {
    // Key not found or error
}
```

**Returns**: String value or error if key doesn't exist

### SET
Set the value of a key with optional TTL.

```go
// Set without expiration
err := client.Set("mykey", "myvalue", 0)

// Set with 60 second TTL
err := client.Set("mykey", "myvalue", 60*time.Second)
```

**Parameters**:
- `key`: String key
- `value`: String value
- `ttl`: Time duration (0 for no expiration)

### DEL
Delete a key.

```go
deleted, err := client.Del("mykey")
// deleted is true if key existed and was deleted
```

**Returns**: Boolean indicating if key was deleted

### EXISTS
Check if a key exists.

```go
exists, err := client.Exists("mykey")
```

**Returns**: Boolean indicating if key exists

## Counter Operations

### INCR
Increment a key's integer value by 1.

```go
newValue, err := client.Incr("counter")
// If key doesn't exist, it's set to 1
```

**Returns**: New integer value after increment

### DECR
Decrement a key's integer value by 1.

```go
newValue, err := client.Decr("counter")
// If key doesn't exist, it's set to -1
```

**Returns**: New integer value after decrement

## Expiration Operations

### EXPIRE
Set a timeout on a key.

```go
success, err := client.Expire("mykey", 30*time.Second)
// success is true if key exists and timeout was set
```

**Parameters**:
- `key`: String key
- `ttl`: Time duration for expiration

**Returns**: Boolean indicating if timeout was set

### TTL
Get the remaining time to live of a key.

```go
ttl, err := client.TTL("mykey")
// ttl is remaining duration, negative values have special meaning:
// -1: key exists but has no expiration
// -2: key does not exist
```

**Returns**: Time duration remaining

## Hash Operations

### HGET
Get the value of a hash field.

```go
value, err := client.HGet("myhash", "field1")
```

**Parameters**:
- `key`: Hash key
- `field`: Field name

**Returns**: Field value or error if not found

### HSET
Set the value of a hash field.

```go
err := client.HSet("myhash", "field1", "value1")
```

**Parameters**:
- `key`: Hash key
- `field`: Field name
- `value`: Field value

### HGETALL
Get all fields and values in a hash.

```go
hash, err := client.HGetAll("myhash")
// hash is map[string]string with all field-value pairs
```

**Returns**: Map of field-value pairs

## List Operations

### LPUSH
Insert elements at the head of a list.

```go
length, err := client.LPush("mylist", "item1", "item2", "item3")
// Items are inserted in reverse order: item3, item2, item1
```

**Parameters**:
- `key`: List key
- `values`: Variable number of string values

**Returns**: New length of the list

### RPUSH
Insert elements at the tail of a list.

```go
length, err := client.RPush("mylist", "item4", "item5")
```

**Parameters**:
- `key`: List key
- `values`: Variable number of string values

**Returns**: New length of the list

### LPOP
Remove and return the first element of a list.

```go
value, err := client.LPop("mylist")
```

**Returns**: First element or error if list is empty

## Set Operations

### SADD
Add members to a set.

```go
added, err := client.SAdd("myset", "member1", "member2", "member3")
// added is the number of new members added (duplicates ignored)
```

**Parameters**:
- `key`: Set key
- `members`: Variable number of string members

**Returns**: Number of members actually added

### SMEMBERS
Get all members of a set.

```go
members, err := client.SMembers("myset")
// members is []string with all set members
```

**Returns**: Slice of all set members

## Utility Operations

### PING
Test connectivity to the cluster.

```go
err := client.Ping()
// err is nil if at least one node is reachable
```

**Returns**: Error if no nodes are reachable

## Error Handling

All client methods return errors for various conditions:

- **Network errors**: Connection failures, timeouts
- **Protocol errors**: Invalid responses, serialization issues
- **Application errors**: Key not found, type mismatches
- **Configuration errors**: Invalid node addresses, bad parameters

```go
value, err := client.Get("mykey")
if err != nil {
    switch {
    case strings.Contains(err.Error(), "key not found"):
        // Handle missing key
    case strings.Contains(err.Error(), "connection"):
        // Handle network issues
    default:
        // Handle other errors
    }
}
```

## Configuration Options

### Environment Variables

- `CACHEMIR_NODES`: Comma-separated list of node addresses
- `CACHEMIR_MAX_CONNS_PER_NODE`: Maximum connections per node (default: 10)
- `CACHEMIR_CONN_TIMEOUT`: Connection timeout in seconds (default: 5)
- `CACHEMIR_READ_TIMEOUT`: Read timeout in seconds (default: 30)
- `CACHEMIR_WRITE_TIMEOUT`: Write timeout in seconds (default: 10)
- `CACHEMIR_RETRY_ATTEMPTS`: Number of retry attempts (default: 3)
- `CACHEMIR_VIRTUAL_NODES`: Virtual nodes for consistent hashing (default: 150)

### Programmatic Configuration

```go
config := &config.ClientConfig{
    Nodes:           []string{"node1:8080", "node2:8080", "node3:8080"},
    MaxConnsPerNode: 20,
    ConnTimeout:     10,
    ReadTimeout:     30,
    WriteTimeout:    10,
    RetryAttempts:   5,
    VirtualNodes:    200,
}

client := client.NewWithConfig(config)
```

## Best Practices

### Connection Management
- Always call `client.Close()` when done
- Reuse client instances across your application
- Configure appropriate connection pool sizes

### Error Handling
- Always check for errors
- Implement retry logic for transient failures
- Use circuit breakers for failing nodes

### Key Design
- Use consistent key naming conventions
- Avoid very long key names
- Consider key distribution across nodes

### Performance
- Use connection pooling effectively
- Batch operations when possible
- Monitor client-side metrics

### Monitoring
- Track error rates and latencies
- Monitor connection pool utilization
- Set up alerts for node failures
