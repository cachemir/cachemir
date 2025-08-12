// Package server implements the CacheMir cache server with TCP networking and command processing.
//
// The server provides a high-performance TCP interface for cache operations,
// handling multiple concurrent connections and executing Redis-compatible commands.
// It uses the binary protocol for efficient communication and integrates with
// the cache engine for data storage.
//
// Architecture:
//   - TCP server with concurrent connection handling
//   - Binary protocol for client-server communication
//   - Integration with cache engine for data operations
//   - Graceful shutdown support
//   - Configurable timeouts and connection limits
//
// Example usage:
//
//	server := server.New(8080)
//	if err := server.Start(); err != nil {
//		log.Fatal(err)
//	}
//
// The server handles all Redis-compatible commands including:
//   - String operations: GET, SET, DEL, EXISTS, INCR, DECR
//   - Expiration: EXPIRE, TTL, PERSIST
//   - Hash operations: HGET, HSET, HDEL, HGETALL
//   - List operations: LPUSH, RPUSH, LPOP, RPOP, LLEN
//   - Set operations: SADD, SREM, SMEMBERS, SISMEMBER
//   - Utility: PING
package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/cachemir/cachemir/pkg/cache"
	"github.com/cachemir/cachemir/pkg/protocol"
)

// Server timeout constants
const (
	defaultReadTimeoutSecs  = 30
	defaultWriteTimeoutSecs = 10
	hashCapacityFactor      = 2
	minHashFields           = 2
)

// Server represents a CacheMir cache server instance.
// It manages TCP connections, processes commands, and maintains the cache state.
// The server is designed to handle multiple concurrent connections efficiently.
//
// Example:
//
//	server := server.New(8080)
//	go func() {
//		if err := server.Start(); err != nil {
//			log.Printf("Server error: %v", err)
//		}
//	}()
//
//	// Later, to stop the server
//	server.Stop()
type Server struct {
	cache    *cache.Cache // The underlying cache engine
	listener net.Listener // TCP listener for incoming connections
	port     int          // Port number to listen on
}

// New creates a new Server instance that will listen on the specified port.
// The server is not started until Start() is called.
//
// Example:
//
//	server := server.New(8080)
//	// Server is created but not yet listening
//
// Parameters:
//   - port: The TCP port number to listen on
//
// Returns:
//   - A new Server instance ready to be started
func New(port int) *Server {
	return &Server{
		cache: cache.New(),
		port:  port,
	}
}

// Start begins listening for TCP connections and processing commands.
// This method blocks until the server is stopped or encounters an error.
// Each incoming connection is handled in a separate goroutine for concurrency.
//
// The server will:
//  1. Create a TCP listener on the configured port
//  2. Accept incoming connections in a loop
//  3. Spawn a goroutine for each connection to handle commands
//  4. Continue until Stop() is called or an error occurs
//
// Example:
//
//	server := server.New(8080)
//	log.Println("Starting server...")
//	if err := server.Start(); err != nil {
//		log.Fatalf("Server failed: %v", err)
//	}
//
// Returns:
//   - Error if the server fails to start or encounters a fatal error
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	lc := net.ListenConfig{}
	listener, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener
	log.Printf("CacheMir server listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

// Stop gracefully shuts down the server by closing the TCP listener.
// This will cause Start() to return and stop accepting new connections.
// Existing connections will continue to be processed until they complete.
//
// Example:
//
//	// In a signal handler or shutdown routine
//	if err := server.Stop(); err != nil {
//		log.Printf("Error stopping server: %v", err)
//	}
//
// Returns:
//   - Error if there was a problem closing the listener
func (s *Server) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// handleConnection processes commands from a single client connection.
// It runs in its own goroutine and handles the complete lifecycle of a connection:
//  1. Read commands from the client using the binary protocol
//  2. Execute each command against the cache
//  3. Send responses back to the client
//  4. Handle connection errors and cleanup
//
// The connection has timeouts for both reading and writing to prevent
// hanging connections from consuming resources.
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}()

	for {
		if err := conn.SetReadDeadline(time.Now().Add(defaultReadTimeoutSecs * time.Second)); err != nil {
			log.Printf("Error setting read deadline: %v", err)
			return
		}

		cmd, err := protocol.ReadCommand(conn)
		if err != nil {
			log.Printf("Failed to read command: %v", err)
			return
		}

		resp := s.executeCommand(cmd)

		if err := conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeoutSecs * time.Second)); err != nil {
			log.Printf("Error setting write deadline: %v", err)
			return
		}
		if err := protocol.WriteResponse(conn, resp); err != nil {
			log.Printf("Failed to write response: %v", err)
			return
		}
	}
}

// executeCommand processes a single command and returns the appropriate response.
// It acts as a dispatcher, routing commands to their specific handler methods
// based on the command type. Unknown commands return an error response.
//
// Parameters:
//   - cmd: The command to execute
//
// Returns:
//   - Response object containing the result or error
func (s *Server) executeCommand(cmd *protocol.Command) *protocol.Response {
	if handler := s.getCommandHandler(cmd.Type); handler != nil {
		return handler(cmd)
	}

	return &protocol.Response{
		Type:  protocol.RespError,
		Error: fmt.Sprintf("unknown command: %d", cmd.Type),
	}
}

func (s *Server) getCommandHandler(cmdType protocol.CommandType) func(*protocol.Command) *protocol.Response {
	handlers := map[protocol.CommandType]func(*protocol.Command) *protocol.Response{
		protocol.CmdGet:       s.handleGet,
		protocol.CmdSet:       s.handleSet,
		protocol.CmdDel:       s.handleDel,
		protocol.CmdExists:    s.handleExists,
		protocol.CmdIncr:      s.handleIncr,
		protocol.CmdDecr:      s.handleDecr,
		protocol.CmdIncrBy:    s.handleIncrBy,
		protocol.CmdDecrBy:    s.handleDecrBy,
		protocol.CmdExpire:    s.handleExpire,
		protocol.CmdTTL:       s.handleTTL,
		protocol.CmdPersist:   s.handlePersist,
		protocol.CmdHGet:      s.handleHGet,
		protocol.CmdHSet:      s.handleHSet,
		protocol.CmdHDel:      s.handleHDel,
		protocol.CmdHExists:   s.handleHExists,
		protocol.CmdHGetAll:   s.handleHGetAll,
		protocol.CmdLPush:     s.handleLPush,
		protocol.CmdRPush:     s.handleRPush,
		protocol.CmdLPop:      s.handleLPop,
		protocol.CmdRPop:      s.handleRPop,
		protocol.CmdLLen:      s.handleLLen,
		protocol.CmdSAdd:      s.handleSAdd,
		protocol.CmdSRem:      s.handleSRem,
		protocol.CmdSMembers:  s.handleSMembers,
		protocol.CmdSIsMember: s.handleSIsMember,
		protocol.CmdPing:      s.handlePing,
	}

	return handlers[cmdType]
}

func (s *Server) handlePing(_ *protocol.Command) *protocol.Response {
	return &protocol.Response{Type: protocol.RespString, Data: "PONG"}
}

// handleGet processes GET commands to retrieve string values.
// Returns the value if found, or a nil response if the key doesn't exist.
func (s *Server) handleGet(cmd *protocol.Command) *protocol.Response {
	value, exists := s.cache.Get(cmd.Key)
	if !exists {
		return &protocol.Response{Type: protocol.RespNil}
	}
	return &protocol.Response{Type: protocol.RespString, Data: value}
}

// handleSet processes SET commands to store string values.
// Uses the TTL from the command if specified.
// Returns an OK response on success, or an error if arguments are invalid.
func (s *Server) handleSet(cmd *protocol.Command) *protocol.Response {
	if len(cmd.Args) == 0 {
		return &protocol.Response{Type: protocol.RespError, Error: "SET requires a value"}
	}
	s.cache.Set(cmd.Key, cmd.Args[0], cmd.TTL)
	return &protocol.Response{Type: protocol.RespOK}
}

// handleDel processes DEL commands to delete keys.
// Returns 1 if the key was deleted, 0 if it didn't exist.
func (s *Server) handleDel(cmd *protocol.Command) *protocol.Response {
	deleted := s.cache.Del(cmd.Key)
	var result int64 = 0
	if deleted {
		result = 1
	}
	return &protocol.Response{Type: protocol.RespInt, Data: result}
}

// handleExists processes EXISTS commands to check key existence.
// Returns 1 if the key exists, 0 if it doesn't.
func (s *Server) handleExists(cmd *protocol.Command) *protocol.Response {
	exists := s.cache.Exists(cmd.Key)
	var result int64 = 0
	if exists {
		result = 1
	}
	return &protocol.Response{Type: protocol.RespInt, Data: result}
}

// handleIncr processes INCR commands to increment integer values.
// Returns the new value after incrementing, or an error if the value is not an integer.
func (s *Server) handleIncr(cmd *protocol.Command) *protocol.Response {
	value, err := s.cache.Incr(cmd.Key)
	if err != nil {
		return &protocol.Response{Type: protocol.RespError, Error: err.Error()}
	}
	return &protocol.Response{Type: protocol.RespInt, Data: value}
}

// handleDecr processes DECR commands to decrement integer values.
// Returns the new value after decrementing, or an error if the value is not an integer.
func (s *Server) handleDecr(cmd *protocol.Command) *protocol.Response {
	value, err := s.cache.Decr(cmd.Key)
	if err != nil {
		return &protocol.Response{Type: protocol.RespError, Error: err.Error()}
	}
	return &protocol.Response{Type: protocol.RespInt, Data: value}
}

// handleIncrBy processes INCRBY commands to increment by a specific amount.
// Parses the delta from the first argument and applies it to the key.
// Returns the new value or an error if the delta is invalid or the value is not an integer.
func (s *Server) handleIncrBy(cmd *protocol.Command) *protocol.Response {
	if len(cmd.Args) == 0 {
		return &protocol.Response{Type: protocol.RespError, Error: "INCRBY requires a delta value"}
	}

	delta := int64(1)
	if len(cmd.Args) > 0 {
		if d, err := parseIntArg(cmd.Args[0]); err == nil {
			delta = d
		}
	}

	value, err := s.cache.IncrBy(cmd.Key, delta)
	if err != nil {
		return &protocol.Response{Type: protocol.RespError, Error: err.Error()}
	}
	return &protocol.Response{Type: protocol.RespInt, Data: value}
}

// handleDecrBy processes DECRBY commands to decrement by a specific amount.
// Parses the delta from the first argument and subtracts it from the key.
// Returns the new value or an error if the delta is invalid or the value is not an integer.
func (s *Server) handleDecrBy(cmd *protocol.Command) *protocol.Response {
	if len(cmd.Args) == 0 {
		return &protocol.Response{Type: protocol.RespError, Error: "DECRBY requires a delta value"}
	}

	delta := int64(1)
	if len(cmd.Args) > 0 {
		if d, err := parseIntArg(cmd.Args[0]); err == nil {
			delta = -d
		}
	}

	value, err := s.cache.IncrBy(cmd.Key, delta)
	if err != nil {
		return &protocol.Response{Type: protocol.RespError, Error: err.Error()}
	}
	return &protocol.Response{Type: protocol.RespInt, Data: value}
}

// handleExpire processes EXPIRE commands to set key expiration.
// Uses the TTL from the command to set the expiration time.
// Returns 1 if the expiration was set, 0 if the key doesn't exist.
func (s *Server) handleExpire(cmd *protocol.Command) *protocol.Response {
	success := s.cache.Expire(cmd.Key, cmd.TTL)
	var result int64 = 0
	if success {
		result = 1
	}
	return &protocol.Response{Type: protocol.RespInt, Data: result}
}

// handleTTL processes TTL commands to get remaining time to live.
// Returns the TTL in seconds, with special values for non-existent keys
// and keys without expiration.
func (s *Server) handleTTL(cmd *protocol.Command) *protocol.Response {
	ttl := s.cache.TTL(cmd.Key)
	return &protocol.Response{Type: protocol.RespInt, Data: int64(ttl.Seconds())}
}

// handlePersist processes PERSIST commands to remove key expiration.
// Returns 1 if the expiration was removed, 0 if the key doesn't exist or has no expiration.
func (s *Server) handlePersist(cmd *protocol.Command) *protocol.Response {
	success := s.cache.Persist(cmd.Key)
	var result int64 = 0
	if success {
		result = 1
	}
	return &protocol.Response{Type: protocol.RespInt, Data: result}
}

// handleHGet processes HGET commands to retrieve hash field values.
// Returns the field value if found, or a nil response if the hash or field doesn't exist.
func (s *Server) handleHGet(cmd *protocol.Command) *protocol.Response {
	if len(cmd.Args) == 0 {
		return &protocol.Response{Type: protocol.RespError, Error: "HGET requires a field"}
	}

	value, exists := s.cache.HGet(cmd.Key, cmd.Args[0])
	if !exists {
		return &protocol.Response{Type: protocol.RespNil}
	}
	return &protocol.Response{Type: protocol.RespString, Data: value}
}

// handleHSet processes HSET commands to set hash field values.
// Requires both field and value arguments.
// Returns an OK response on success, or an error if arguments are missing.
func (s *Server) handleHSet(cmd *protocol.Command) *protocol.Response {
	if len(cmd.Args) < minHashFields {
		return &protocol.Response{Type: protocol.RespError, Error: "HSET requires field and value"}
	}

	s.cache.HSet(cmd.Key, cmd.Args[0], cmd.Args[1])
	return &protocol.Response{Type: protocol.RespOK}
}

// handleHDel processes HDEL commands to delete hash fields.
// Returns 1 if the field was deleted, 0 if it didn't exist.
func (s *Server) handleHDel(cmd *protocol.Command) *protocol.Response {
	if len(cmd.Args) == 0 {
		return &protocol.Response{Type: protocol.RespError, Error: "HDEL requires a field"}
	}

	deleted := s.cache.HDel(cmd.Key, cmd.Args[0])
	var result int64 = 0
	if deleted {
		result = 1
	}
	return &protocol.Response{Type: protocol.RespInt, Data: result}
}

// handleHExists processes HEXISTS commands to check if a hash field exists.
// Returns 1 if the field exists, 0 otherwise.
func (s *Server) handleHExists(cmd *protocol.Command) *protocol.Response {
	if len(cmd.Args) == 0 {
		return &protocol.Response{Type: protocol.RespError, Error: "HEXISTS requires a field"}
	}

	exists := s.cache.HExists(cmd.Key, cmd.Args[0])
	var result int64 = 0
	if exists {
		result = 1
	}
	return &protocol.Response{Type: protocol.RespInt, Data: result}
}

// handleHGetAll processes HGETALL commands to retrieve all hash fields and values.
// Returns an array containing alternating field names and values.
func (s *Server) handleHGetAll(cmd *protocol.Command) *protocol.Response {
	hash := s.cache.HGetAll(cmd.Key)
	result := make([]string, 0, len(hash)*hashCapacityFactor)
	for k, v := range hash {
		result = append(result, k, v)
	}
	return &protocol.Response{Type: protocol.RespArray, Data: result}
}

// handleLPush processes LPUSH commands to add elements to the head of a list.
// Returns the new length of the list after insertion.
func (s *Server) handleLPush(cmd *protocol.Command) *protocol.Response {
	if len(cmd.Args) == 0 {
		return &protocol.Response{Type: protocol.RespError, Error: "LPUSH requires at least one value"}
	}

	length := s.cache.LPush(cmd.Key, cmd.Args...)
	return &protocol.Response{Type: protocol.RespInt, Data: int64(length)}
}

// handleRPush processes RPUSH commands to add elements to the tail of a list.
// Returns the new length of the list after insertion.
func (s *Server) handleRPush(cmd *protocol.Command) *protocol.Response {
	if len(cmd.Args) == 0 {
		return &protocol.Response{Type: protocol.RespError, Error: "RPUSH requires at least one value"}
	}

	length := s.cache.RPush(cmd.Key, cmd.Args...)
	return &protocol.Response{Type: protocol.RespInt, Data: int64(length)}
}

// handleLPop processes LPOP commands to remove elements from the head of a list.
// Returns the removed element, or a nil response if the list is empty or doesn't exist.
func (s *Server) handleLPop(cmd *protocol.Command) *protocol.Response {
	value, exists := s.cache.LPop(cmd.Key)
	if !exists {
		return &protocol.Response{Type: protocol.RespNil}
	}
	return &protocol.Response{Type: protocol.RespString, Data: value}
}

// handleRPop processes RPOP commands to remove elements from the tail of a list.
// Returns the removed element, or a nil response if the list is empty or doesn't exist.
func (s *Server) handleRPop(cmd *protocol.Command) *protocol.Response {
	value, exists := s.cache.RPop(cmd.Key)
	if !exists {
		return &protocol.Response{Type: protocol.RespNil}
	}
	return &protocol.Response{Type: protocol.RespString, Data: value}
}

// handleLLen processes LLEN commands to get the length of a list.
// Returns the number of elements in the list, or 0 if the list doesn't exist.
func (s *Server) handleLLen(cmd *protocol.Command) *protocol.Response {
	length := s.cache.LLen(cmd.Key)
	return &protocol.Response{Type: protocol.RespInt, Data: int64(length)}
}

// handleSAdd processes SADD commands to add members to a set.
// Returns the number of members that were actually added (excluding duplicates).
func (s *Server) handleSAdd(cmd *protocol.Command) *protocol.Response {
	if len(cmd.Args) == 0 {
		return &protocol.Response{Type: protocol.RespError, Error: "SADD requires at least one member"}
	}

	added := s.cache.SAdd(cmd.Key, cmd.Args...)
	return &protocol.Response{Type: protocol.RespInt, Data: int64(added)}
}

// handleSRem processes SREM commands to remove members from a set.
// Returns the number of members that were actually removed.
func (s *Server) handleSRem(cmd *protocol.Command) *protocol.Response {
	if len(cmd.Args) == 0 {
		return &protocol.Response{Type: protocol.RespError, Error: "SREM requires at least one member"}
	}

	removed := s.cache.SRem(cmd.Key, cmd.Args...)
	return &protocol.Response{Type: protocol.RespInt, Data: int64(removed)}
}

// handleSMembers processes SMEMBERS commands to get all members of a set.
// Returns an array containing all set members.
func (s *Server) handleSMembers(cmd *protocol.Command) *protocol.Response {
	members := s.cache.SMembers(cmd.Key)
	return &protocol.Response{Type: protocol.RespArray, Data: members}
}

// handleSIsMember processes SISMEMBER commands to check set membership.
// Returns 1 if the member exists in the set, 0 otherwise.
func (s *Server) handleSIsMember(cmd *protocol.Command) *protocol.Response {
	if len(cmd.Args) == 0 {
		return &protocol.Response{Type: protocol.RespError, Error: "SISMEMBER requires a member"}
	}

	isMember := s.cache.SIsMember(cmd.Key, cmd.Args[0])
	var result int64 = 0
	if isMember {
		result = 1
	}
	return &protocol.Response{Type: protocol.RespInt, Data: result}
}

// parseIntArg parses a string argument as a 64-bit signed integer.
// This is used for commands that require integer arguments like INCRBY and DECRBY.
//
// Parameters:
//   - arg: String representation of an integer
//
// Returns:
//   - Parsed integer value
//   - Error if the string is not a valid integer
func parseIntArg(arg string) (int64, error) {
	return strconv.ParseInt(arg, 10, 64)
}
