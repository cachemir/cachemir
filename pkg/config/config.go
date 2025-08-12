// Package config provides configuration management for CacheMir server and client components.
//
// The package supports configuration through multiple sources with the following precedence:
//  1. Command-line flags (highest priority)
//  2. Environment variables
//  3. Default values (lowest priority)
//
// Server Configuration:
//   - Port and host binding settings
//   - Connection limits and timeouts
//   - Logging configuration
//   - Resource constraints
//
// Client Configuration:
//   - Node discovery and connection settings
//   - Connection pooling parameters
//   - Retry policies and timeouts
//   - Consistent hashing parameters
//
// Example server usage:
//
//	config := config.LoadServerConfig()
//	if err := config.Validate(); err != nil {
//		log.Fatal(err)
//	}
//	server := server.New(config.Port)
//
// Example client usage:
//
//	config := config.LoadClientConfig()
//	config.Nodes = []string{"server1:8080", "server2:8080"}
//	client := client.NewWithConfig(config)
//
// Environment variables are prefixed with "CACHEMIR_" and use uppercase names.
// For example, the server port can be set with CACHEMIR_PORT=8080.
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Default server configuration constants
const (
	DefaultServerPort         = 8080
	DefaultMaxConnections     = 1000
	DefaultReadTimeoutSecs    = 30
	DefaultWriteTimeoutSecs   = 10
	DefaultMaxConnsPerNode    = 10
	DefaultConnTimeoutSecs    = 5
	DefaultRetryAttempts      = 3
	DefaultVirtualNodes       = 150
	DefaultHashCapacityFactor = 2
)

// Protocol constants
const (
	ProtocolHeaderSize = 4
	MaxUint32Value     = 4294967295
)

// ServerConfig holds all configuration options for a CacheMir server instance.
// It includes network settings, resource limits, and operational parameters.
//
// Configuration sources (in order of precedence):
//  1. Command-line flags: -port, -host, -max-conns, etc.
//  2. Environment variables: CACHEMIR_PORT, CACHEMIR_HOST, etc.
//  3. Default values
//
// Example:
//
//	config := &ServerConfig{
//		Port:     8080,
//		Host:     "0.0.0.0",
//		MaxConns: 1000,
//	}
//	if err := config.Validate(); err != nil {
//		log.Fatal(err)
//	}
type ServerConfig struct {
	Host         string // Host address to bind to (default: "0.0.0.0")
	LogLevel     string // Log level: debug, info, warn, error (default: "info")
	Port         int    // TCP port to listen on (default: 8080)
	MaxConns     int    // Maximum concurrent connections (default: 1000)
	ReadTimeout  int    // Read timeout in seconds (default: 30)
	WriteTimeout int    // Write timeout in seconds (default: 10)
}

// ClientConfig holds all configuration options for a CacheMir client instance.
// It includes node discovery, connection pooling, and retry settings.
//
// Configuration sources (in order of precedence):
//  1. Programmatic configuration
//  2. Environment variables: CACHEMIR_NODES, CACHEMIR_MAX_CONNS_PER_NODE, etc.
//  3. Default values
//
// Example:
//
//	config := &ClientConfig{
//		Nodes:           []string{"server1:8080", "server2:8080"},
//		MaxConnsPerNode: 20,
//		RetryAttempts:   3,
//	}
//	client := client.NewWithConfig(config)
type ClientConfig struct {
	Nodes           []string // List of server addresses (default: ["localhost:8080"])
	MaxConnsPerNode int      // Max connections per server node (default: 10)
	ConnTimeout     int      // Connection timeout in seconds (default: 5)
	ReadTimeout     int      // Read timeout in seconds (default: 30)
	WriteTimeout    int      // Write timeout in seconds (default: 10)
	RetryAttempts   int      // Number of retry attempts (default: 3)
	VirtualNodes    int      // Virtual nodes for consistent hashing (default: 150)
}

// LoadServerConfig creates a ServerConfig by loading values from command-line flags
// and environment variables, with sensible defaults.
//
// Command-line flags:
//
//	-port: Server port (default: 8080)
//	-host: Server host (default: "0.0.0.0")
//	-max-conns: Maximum connections (default: 1000)
//	-read-timeout: Read timeout in seconds (default: 30)
//	-write-timeout: Write timeout in seconds (default: 10)
//	-log-level: Log level (default: "info")
//
// Environment variables:
//
//	CACHEMIR_PORT: Server port
//	CACHEMIR_HOST: Server host
//	CACHEMIR_MAX_CONNS: Maximum connections
//
// Example:
//
//	// Load configuration from flags and environment
//	config := config.LoadServerConfig()
//
//	// Validate the configuration
//	if err := config.Validate(); err != nil {
//		log.Fatalf("Invalid configuration: %v", err)
//	}
//
//	// Use the configuration
//	server := server.New(config.Port)
//
// Returns:
//   - ServerConfig with values loaded from various sources
func LoadServerConfig() *ServerConfig {
	config := &ServerConfig{
		Port:         DefaultServerPort,
		Host:         "0.0.0.0",
		MaxConns:     DefaultMaxConnections,
		ReadTimeout:  DefaultReadTimeoutSecs,
		WriteTimeout: DefaultWriteTimeoutSecs,
		LogLevel:     "info",
	}

	flag.IntVar(&config.Port, "port", config.Port, "Server port")
	flag.StringVar(&config.Host, "host", config.Host, "Server host")
	flag.IntVar(&config.MaxConns, "max-conns", config.MaxConns, "Maximum concurrent connections")
	flag.IntVar(&config.ReadTimeout, "read-timeout", config.ReadTimeout, "Read timeout in seconds")
	flag.IntVar(&config.WriteTimeout, "write-timeout", config.WriteTimeout, "Write timeout in seconds")
	flag.StringVar(&config.LogLevel, "log-level", config.LogLevel, "Log level (debug, info, warn, error)")
	flag.Parse()

	if port := os.Getenv("CACHEMIR_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	}

	if host := os.Getenv("CACHEMIR_HOST"); host != "" {
		config.Host = host
	}

	if maxConns := os.Getenv("CACHEMIR_MAX_CONNS"); maxConns != "" {
		if mc, err := strconv.Atoi(maxConns); err == nil {
			config.MaxConns = mc
		}
	}

	return config
}

// LoadClientConfig creates a ClientConfig by loading values from environment
// variables, with sensible defaults.
//
// Environment variables:
//
//	CACHEMIR_NODES: Comma-separated list of server addresses
//	CACHEMIR_MAX_CONNS_PER_NODE: Maximum connections per server
//	CACHEMIR_CONN_TIMEOUT: Connection timeout in seconds
//	CACHEMIR_READ_TIMEOUT: Read timeout in seconds
//	CACHEMIR_WRITE_TIMEOUT: Write timeout in seconds
//	CACHEMIR_RETRY_ATTEMPTS: Number of retry attempts
//	CACHEMIR_VIRTUAL_NODES: Virtual nodes for consistent hashing
//
// Example:
//
//	// Set environment variables
//	os.Setenv("CACHEMIR_NODES", "server1:8080,server2:8080,server3:8080")
//	os.Setenv("CACHEMIR_MAX_CONNS_PER_NODE", "20")
//
//	// Load configuration
//	config := config.LoadClientConfig()
//
//	// Validate and use
//	if err := config.Validate(); err != nil {
//		log.Fatal(err)
//	}
//	client := client.NewWithConfig(config)
//
// Returns:
//   - ClientConfig with values loaded from environment variables and defaults
func LoadClientConfig() *ClientConfig {
	config := &ClientConfig{
		Nodes:           []string{"localhost:8080"},
		MaxConnsPerNode: DefaultMaxConnsPerNode,
		ConnTimeout:     DefaultConnTimeoutSecs,
		ReadTimeout:     DefaultReadTimeoutSecs,
		WriteTimeout:    DefaultWriteTimeoutSecs,
		RetryAttempts:   DefaultRetryAttempts,
		VirtualNodes:    DefaultVirtualNodes,
	}

	if nodes := os.Getenv("CACHEMIR_NODES"); nodes != "" {
		config.Nodes = strings.Split(nodes, ",")
		for i, node := range config.Nodes {
			config.Nodes[i] = strings.TrimSpace(node)
		}
	}

	if maxConns := os.Getenv("CACHEMIR_MAX_CONNS_PER_NODE"); maxConns != "" {
		if mc, err := strconv.Atoi(maxConns); err == nil {
			config.MaxConnsPerNode = mc
		}
	}

	if connTimeout := os.Getenv("CACHEMIR_CONN_TIMEOUT"); connTimeout != "" {
		if ct, err := strconv.Atoi(connTimeout); err == nil {
			config.ConnTimeout = ct
		}
	}

	if readTimeout := os.Getenv("CACHEMIR_READ_TIMEOUT"); readTimeout != "" {
		if rt, err := strconv.Atoi(readTimeout); err == nil {
			config.ReadTimeout = rt
		}
	}

	if writeTimeout := os.Getenv("CACHEMIR_WRITE_TIMEOUT"); writeTimeout != "" {
		if wt, err := strconv.Atoi(writeTimeout); err == nil {
			config.WriteTimeout = wt
		}
	}

	if retryAttempts := os.Getenv("CACHEMIR_RETRY_ATTEMPTS"); retryAttempts != "" {
		if ra, err := strconv.Atoi(retryAttempts); err == nil {
			config.RetryAttempts = ra
		}
	}

	if virtualNodes := os.Getenv("CACHEMIR_VIRTUAL_NODES"); virtualNodes != "" {
		if vn, err := strconv.Atoi(virtualNodes); err == nil {
			config.VirtualNodes = vn
		}
	}

	return config
}

// Address returns the full address string for the server to bind to.
// It combines the host and port into a format suitable for net.Listen().
//
// Example:
//
//	config := &ServerConfig{Host: "0.0.0.0", Port: 8080}
//	addr := config.Address() // Returns "0.0.0.0:8080"
//
// Returns:
//   - Address string in "host:port" format
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Validate checks if the ServerConfig contains valid values.
// It verifies that all numeric values are within acceptable ranges
// and that string values are from valid sets.
//
// Validation rules:
//   - Port must be between 1 and 65535
//   - MaxConns must be positive
//   - ReadTimeout must be positive
//   - WriteTimeout must be positive
//   - LogLevel must be one of: debug, info, warn, error
//
// Example:
//
//	config := config.LoadServerConfig()
//	if err := config.Validate(); err != nil {
//		log.Fatalf("Configuration error: %v", err)
//	}
//
// Returns:
//   - nil if configuration is valid
//   - Error describing the first validation failure found
func (c *ServerConfig) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.MaxConns < 1 {
		return fmt.Errorf("max connections must be positive: %d", c.MaxConns)
	}

	if c.ReadTimeout < 1 {
		return fmt.Errorf("read timeout must be positive: %d", c.ReadTimeout)
	}

	if c.WriteTimeout < 1 {
		return fmt.Errorf("write timeout must be positive: %d", c.WriteTimeout)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.LogLevel)
	}

	return nil
}

// Validate checks if the ClientConfig contains valid values.
// It verifies that all settings are within acceptable ranges and
// that required fields are properly configured.
//
// Validation rules:
//   - At least one node must be specified
//   - All node addresses must be non-empty and contain a colon
//   - MaxConnsPerNode must be positive
//   - All timeout values must be positive
//   - RetryAttempts must be non-negative
//   - VirtualNodes must be positive
//
// Example:
//
//	config := &ClientConfig{
//		Nodes: []string{"server1:8080", "server2:8080"},
//		MaxConnsPerNode: 10,
//		RetryAttempts: 3,
//	}
//	if err := config.Validate(); err != nil {
//		log.Fatalf("Configuration error: %v", err)
//	}
//
// Returns:
//   - nil if configuration is valid
//   - Error describing the first validation failure found
func (c *ClientConfig) Validate() error {
	if len(c.Nodes) == 0 {
		return fmt.Errorf("at least one node must be specified")
	}

	for _, node := range c.Nodes {
		if node == "" {
			return fmt.Errorf("empty node address")
		}
		if !strings.Contains(node, ":") {
			return fmt.Errorf("invalid node address format: %s", node)
		}
	}

	if c.MaxConnsPerNode < 1 {
		return fmt.Errorf("max connections per node must be positive: %d", c.MaxConnsPerNode)
	}

	if c.ConnTimeout < 1 {
		return fmt.Errorf("connection timeout must be positive: %d", c.ConnTimeout)
	}

	if c.ReadTimeout < 1 {
		return fmt.Errorf("read timeout must be positive: %d", c.ReadTimeout)
	}

	if c.WriteTimeout < 1 {
		return fmt.Errorf("write timeout must be positive: %d", c.WriteTimeout)
	}

	if c.RetryAttempts < 0 {
		return fmt.Errorf("retry attempts must be non-negative: %d", c.RetryAttempts)
	}

	if c.VirtualNodes < 1 {
		return fmt.Errorf("virtual nodes must be positive: %d", c.VirtualNodes)
	}

	return nil
}
