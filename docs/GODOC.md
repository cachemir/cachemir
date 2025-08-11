# CacheMir GoDoc Documentation

CacheMir now includes comprehensive GoDoc documentation for all packages and exported functions. This guide explains how to view and use the documentation.

## Viewing Documentation

### Local GoDoc Server

Start a local documentation server to browse the full documentation:

```bash
# Install godoc if not already installed
go install golang.org/x/tools/cmd/godoc@latest

# Start the documentation server
godoc -http=:6060

# Open in browser
open http://localhost:6060/pkg/github.com/cachemir/cachemir/
```

### Command Line Documentation

View documentation directly in the terminal:

```bash
# Package overview
go doc github.com/cachemir/cachemir/pkg/client

# Specific function documentation
go doc github.com/cachemir/cachemir/pkg/client.New

# All exported functions in a package
go doc -all github.com/cachemir/cachemir/pkg/cache
```

### Online Documentation

If the project is published to a public repository, documentation will be available at:
- `https://pkg.go.dev/github.com/cachemir/cachemir`

## Documentation Structure

### Package-Level Documentation

Each package includes comprehensive package-level documentation explaining:
- Purpose and use cases
- Architecture and design decisions
- Usage examples
- Integration patterns
- Performance characteristics

### Function Documentation

All exported functions include:
- Clear description of purpose
- Parameter explanations
- Return value descriptions
- Usage examples
- Error conditions
- Thread safety notes where relevant

### Type Documentation

All exported types include:
- Purpose and usage
- Field descriptions
- Method documentation
- Example usage patterns

## Key Documentation Sections

### Client SDK (`pkg/client`)

The client documentation includes:
- Connection management and pooling
- Consistent hashing and node selection
- Redis-compatible API reference
- Error handling strategies
- Configuration options
- Performance tuning

Example viewing client documentation:
```bash
go doc -all github.com/cachemir/cachemir/pkg/client
```

### Cache Engine (`pkg/cache`)

The cache documentation covers:
- Data type support (strings, hashes, lists, sets)
- Expiration and TTL management
- Thread safety guarantees
- Memory management
- Performance characteristics

### Protocol (`pkg/protocol`)

The protocol documentation explains:
- Binary protocol format
- Command and response structures
- Serialization/deserialization
- Network framing
- Error handling

### Consistent Hashing (`pkg/hash`)

The hashing documentation describes:
- Consistent hashing algorithm
- Virtual nodes concept
- Key distribution
- Node addition/removal
- Performance implications

### Configuration (`pkg/config`)

The configuration documentation details:
- Server configuration options
- Client configuration options
- Environment variable support
- Validation rules
- Default values

## Documentation Examples

### Basic Client Usage

```go
// Package client provides a high-level client SDK for connecting to CacheMir cache servers.
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
```

### Advanced Configuration

```go
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
```

## Documentation Standards

The CacheMir documentation follows Go documentation conventions:

1. **Package Comments**: Start with "Package [name] provides..."
2. **Function Comments**: Start with the function name
3. **Examples**: Include practical usage examples
4. **Parameters**: Document all parameters and return values
5. **Errors**: Explain error conditions and handling
6. **Thread Safety**: Note concurrent usage safety
7. **Performance**: Include performance characteristics where relevant

## Generating Documentation

### HTML Documentation

Generate static HTML documentation:

```bash
# Generate documentation for all packages
godoc -html github.com/cachemir/cachemir > docs/cachemir.html

# Generate documentation for specific package
godoc -html github.com/cachemir/cachemir/pkg/client > docs/client.html
```

### Markdown Documentation

Extract documentation as markdown (requires additional tools):

```bash
# Using godoc2md (install separately)
go install github.com/davecheney/godoc2md@latest
godoc2md github.com/cachemir/cachemir/pkg/client > docs/client.md
```

## Documentation Maintenance

### Adding New Documentation

When adding new exported functions or types:

1. Add comprehensive godoc comments
2. Include usage examples
3. Document parameters and return values
4. Explain error conditions
5. Note thread safety implications

### Documentation Review

Before releases:

1. Review all package documentation for accuracy
2. Ensure examples are up-to-date
3. Verify links and references work
4. Test documentation generation
5. Update README with any new features

## Integration with IDEs

Most Go IDEs automatically display godoc documentation:

- **VS Code**: Hover over functions to see documentation
- **GoLand**: Built-in documentation viewer
- **Vim/Neovim**: Use vim-go plugin for documentation
- **Emacs**: Use go-mode for documentation display

## Best Practices

1. **Keep Examples Current**: Ensure code examples compile and work
2. **Be Comprehensive**: Document all exported APIs
3. **Use Clear Language**: Write for developers unfamiliar with the code
4. **Include Context**: Explain why, not just what
5. **Link Related Functions**: Reference related functionality
6. **Update Regularly**: Keep documentation in sync with code changes

The comprehensive godoc documentation makes CacheMir easy to understand, integrate, and maintain for both new and experienced developers.
