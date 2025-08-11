# CacheMir - Distributed In-Memory Cache

A high-performance, horizontally scalable in-memory caching solution with Redis-compatible commands and client-side consistent hashing.

## Features

- **Redis-compatible commands**: GET, SET, DEL, INCR, EXPIRE, HGET, HSET, LPUSH, SADD, etc.
- **Horizontal scaling**: Client-side consistent hashing for automatic node selection
- **Lightweight protocol**: Binary protocol for optimal performance
- **Memory-only**: Fast in-memory storage with no persistence
- **Connection pooling**: Efficient connection management
- **No inter-node sync**: Simple architecture with client-side sharding

## Architecture

```
Client SDK (with consistent hashing)
    ↓
Multiple CacheMir Server Nodes
    ↓
In-Memory Cache Storage
```

## Quick Start

### Server
```bash
go run cmd/server/main.go -port 8080
```

### Client
```go
import "github.com/cachemir/cachemir/pkg/client"

client := client.New([]string{"localhost:8080", "localhost:8081"})
client.Set("key", "value", 0)
value, _ := client.Get("key")
```

## Commands Supported

### String Operations
- GET, SET, DEL, EXISTS
- INCR, DECR, INCRBY, DECRBY

### Expiration
- EXPIRE, TTL, PERSIST

### Hash Operations  
- HGET, HSET, HDEL, HGETALL, HEXISTS

### List Operations
- LPUSH, RPUSH, LPOP, RPOP, LLEN

### Set Operations
- SADD, SREM, SMEMBERS, SISMEMBER

## Documentation

CacheMir provides comprehensive documentation covering all aspects of the system:

### 📚 **Complete Documentation Guide**

- **[Getting Started Guide](GETTING_STARTED.md)** - Quick setup and basic usage
- **[Architecture Documentation](docs/ARCHITECTURE.md)** - System design, scaling, and deployment
- **[API Reference](docs/API.md)** - Complete client SDK API with examples
- **[GoDoc Guide](docs/GODOC.md)** - How to use the comprehensive code documentation

### 🔧 **Code Documentation (GoDoc)**

CacheMir includes comprehensive GoDoc documentation for all packages:

```bash
# Start local documentation server
godoc -http=:6060
# Then visit: http://localhost:6060/pkg/github.com/cachemir/cachemir/

# Or view in terminal
go doc -all github.com/cachemir/cachemir/pkg/client
```

**Package Documentation:**
- **[pkg/client](pkg/client/)** - Client SDK with consistent hashing and connection pooling
- **[pkg/cache](pkg/cache/)** - In-memory cache engine with Redis-like operations  
- **[pkg/protocol](pkg/protocol/)** - Lightweight binary protocol implementation
- **[pkg/hash](pkg/hash/)** - Consistent hashing algorithm with virtual nodes
- **[pkg/config](pkg/config/)** - Configuration management for server and client
- **[internal/server](internal/server/)** - Server implementation with TCP handling

### 💡 **Examples and Usage**

- **[Client Example](cmd/client-example/main.go)** - Comprehensive client usage demonstration
- **[Simple Usage](examples/simple_usage.go)** - Basic operations and patterns
- **[Server Example](cmd/server/main.go)** - Server setup and configuration

### 🏗️ **Development and Building**

- **[Makefile](Makefile)** - Build targets and development commands
- **[Dockerfile](Dockerfile)** - Container deployment configuration


| Topic | File | Description |
|-------|------|-------------|
| **Getting Started** | [GETTING_STARTED.md](GETTING_STARTED.md) | Setup, basic usage, and examples |
| **System Architecture** | [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Design, scaling, and deployment |
| **API Reference** | [docs/API.md](docs/API.md) | Complete client SDK documentation |
| **Code Documentation** | [docs/GODOC.md](docs/GODOC.md) | GoDoc usage and standards |
| **Client Example** | [cmd/client-example/main.go](cmd/client-example/main.go) | Full-featured client demo |
| **Simple Usage** | [examples/simple_usage.go](examples/simple_usage.go) | Basic usage patterns |


## License

MIT License

## 🚀 CI/CD & Automation

CacheMir includes comprehensive GitHub Actions automation for quality assurance and deployment:

### ✅ **Automated Testing**
- **Multi-platform Testing**: Linux, macOS, Windows
- **Multi-version Testing**: Go 1.20, 1.21, 1.22
- **Comprehensive Coverage**: Unit tests, race detection, benchmarks
- **Code Quality**: golangci-lint with 30+ linters
- **Security Scanning**: CodeQL, Gosec, Trivy vulnerability scanning

### 🔄 **Automated Releases**
- **Semantic Versioning**: Based on git tags
- **Multi-platform Binaries**: All major OS/architecture combinations
- **Container Images**: Multi-arch Docker builds pushed to GitHub Container Registry
- **Release Notes**: Auto-generated from commit history
- **GitHub Releases**: Automated release creation with artifacts

### 💬 **PR Automation**
- **Automated Comments**: Test results, coverage, and benchmarks in PR comments
- **Quality Gates**: Comprehensive checks before merge approval
- **Security Alerts**: Vulnerability scanning results
- **Performance Tracking**: Benchmark regression detection

### 🔒 **Security & Compliance**
- **Vulnerability Scanning**: Dependencies and container images
- **Code Analysis**: Static security analysis with CodeQL
- **Dependency Updates**: Automated Dependabot updates
- **SARIF Integration**: Security findings in GitHub Security tab

### 📊 **Quality Metrics**
- **Test Coverage**: Tracked with Codecov integration
- **Code Quality**: Comprehensive linting and static analysis
- **Performance**: Benchmark trends and regression detection
- **Build Health**: Continuous monitoring of CI/CD pipeline

### 🐳 **Container Registry**
```bash
# Pull latest image
docker pull ghcr.io/cachemir/cachemir:latest

# Run container
docker run -p 8080:8080 ghcr.io/cachemir/cachemir:latest
```

### 📋 **Development Workflow**
1. **Fork & Branch**: Create feature branch from main
2. **Develop & Test**: Implement changes with comprehensive tests
3. **Create PR**: Submit pull request with automated CI validation
4. **Code Review**: Maintainer review with automated quality checks
5. **Merge & Release**: Automated release on tag creation

See **[CI/CD Documentation](docs/CICD.md)** for detailed information about the automation pipeline.
