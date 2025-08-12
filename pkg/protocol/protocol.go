// Package protocol implements the lightweight binary protocol for CacheMir client-server communication.
//
// The protocol is designed for efficiency and simplicity, using binary encoding
// to minimize network overhead. It supports all Redis-compatible commands and
// provides structured request/response handling.
//
// Protocol Format:
//   - All messages are prefixed with a 4-byte length header (big-endian)
//   - Commands and responses are binary-encoded using variable-length encoding
//   - Strings are length-prefixed to handle arbitrary data
//
// Example usage:
//
//	// Create a command
//	cmd := &protocol.Command{
//		Type: protocol.CmdSet,
//		Key:  "user:123",
//		Args: []string{"john_doe"},
//		TTL:  time.Hour,
//	}
//
//	// Serialize and send
//	data, err := cmd.Serialize()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Write to connection
//	err = protocol.WriteCommand(conn, cmd)
//
// The protocol supports the following command types:
//   - String operations: GET, SET, DEL, EXISTS, INCR, DECR
//   - Expiration: EXPIRE, TTL, PERSIST
//   - Hash operations: HGET, HSET, HDEL, HGETALL, HEXISTS
//   - List operations: LPUSH, RPUSH, LPOP, RPOP, LLEN
//   - Set operations: SADD, SREM, SMEMBERS, SISMEMBER
//   - Utility: PING
package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Protocol constants
const (
	protocolHeaderSize = 4
	maxUint32Value     = 4294967295
	maxInt64Value      = 9223372036854775807
	minArgsForSet      = 3
	exactArgsForGet    = 2
	exactArgsForDel    = 2
	exactArgsForExists = 2
	exactArgsForIncr   = 2
	exactArgsForDecr   = 2
)

// CommandType represents the type of command being executed.
// Each command type corresponds to a Redis-compatible operation.
type CommandType uint8

// Command type constants define all supported cache operations.
// These match Redis command semantics for compatibility.
const (
	CmdGet       CommandType = iota // GET key - retrieve string value
	CmdSet                          // SET key value [ttl] - store string value
	CmdDel                          // DEL key - delete key
	CmdExists                       // EXISTS key - check if key exists
	CmdIncr                         // INCR key - increment integer value
	CmdDecr                         // DECR key - decrement integer value
	CmdIncrBy                       // INCRBY key delta - increment by delta
	CmdDecrBy                       // DECRBY key delta - decrement by delta
	CmdExpire                       // EXPIRE key ttl - set key expiration
	CmdTTL                          // TTL key - get time to live
	CmdPersist                      // PERSIST key - remove expiration
	CmdHGet                         // HGET key field - get hash field
	CmdHSet                         // HSET key field value - set hash field
	CmdHDel                         // HDEL key field - delete hash field
	CmdHGetAll                      // HGETALL key - get all hash fields
	CmdHExists                      // HEXISTS key field - check hash field exists
	CmdLPush                        // LPUSH key value... - push to list head
	CmdRPush                        // RPUSH key value... - push to list tail
	CmdLPop                         // LPOP key - pop from list head
	CmdRPop                         // RPOP key - pop from list tail
	CmdLLen                         // LLEN key - get list length
	CmdSAdd                         // SADD key member... - add to set
	CmdSRem                         // SREM key member... - remove from set
	CmdSMembers                     // SMEMBERS key - get all set members
	CmdSIsMember                    // SISMEMBER key member - check set membership
	CmdPing                         // PING - connectivity test
)

// ResponseType represents the type of response from the server.
// Different response types carry different data formats.
type ResponseType uint8

// Response type constants define the possible server response formats.
const (
	RespOK     ResponseType = iota // Simple OK response
	RespError                      // Error message response
	RespString                     // String data response
	RespInt                        // Integer data response
	RespArray                      // Array of strings response
	RespNil                        // Null/empty response
)

// Command represents a client request to the cache server.
// It encapsulates the operation type, target key, arguments, and optional TTL.
//
// Example:
//
//	cmd := &Command{
//		Type: CmdSet,
//		Key:  "session:abc123",
//		Args: []string{"user_data"},
//		TTL:  30 * time.Minute,
//	}
type Command struct {
	Key  string        // The target key for the operation
	TTL  time.Duration // Optional time-to-live for expiration
	Type CommandType   // The operation to perform
	Args []string      // Command arguments (values, fields, etc.)
}

// Response represents a server response to a client command.
// The response type determines how the Data field should be interpreted.
//
// Example:
//
//	resp := &Response{
//		Type: RespString,
//		Data: "hello world",
//	}
type Response struct {
	Data  interface{}  // The response payload (string, int64, []string, etc.)
	Error string       // Error message if Type is RespError
	Type  ResponseType // The type of response data
}

// Serialize converts a Command into its binary representation for network transmission.
// The format uses variable-length encoding for efficiency:
//   - 1 byte: command type
//   - varint: key length + key bytes
//   - varint: args count + (varint: arg length + arg bytes) for each arg
//   - varint: TTL in seconds
//
// Example:
//
//	cmd := &Command{Type: CmdGet, Key: "mykey"}
//	data, err := cmd.Serialize()
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Returns:
//   - Binary representation of the command
//   - Error if serialization fails
func (c *Command) Serialize() ([]byte, error) {
	var buf []byte

	buf = append(buf, byte(c.Type))

	keyBytes := []byte(c.Key)
	buf = binary.AppendUvarint(buf, uint64(len(keyBytes)))
	buf = append(buf, keyBytes...)

	buf = binary.AppendUvarint(buf, uint64(len(c.Args)))
	for _, arg := range c.Args {
		argBytes := []byte(arg)
		buf = binary.AppendUvarint(buf, uint64(len(argBytes)))
		buf = append(buf, argBytes...)
	}

	buf = binary.AppendUvarint(buf, uint64(c.TTL.Seconds()))

	return buf, nil
}

// DeserializeCommand reconstructs a Command from its binary representation.
// This is the inverse operation of Command.Serialize().
//
// Parameters:
//   - data: Binary data containing the serialized command
//
// Returns:
//   - Reconstructed Command object
//   - Error if deserialization fails or data is corrupted
//
// Example:
//
//	cmd, err := protocol.DeserializeCommand(data)
//	if err != nil {
//		log.Printf("Failed to deserialize command: %v", err)
//		return
//	}
func DeserializeCommand(data []byte) (*Command, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty command data")
	}

	cmd := &Command{}
	offset := 0

	cmd.Type = CommandType(data[offset])
	offset++

	var err error
	cmd.Key, offset, err = deserializeString(data, offset, "key")
	if err != nil {
		return nil, err
	}

	cmd.Args, offset, err = deserializeStringSlice(data, offset)
	if err != nil {
		return nil, err
	}

	cmd.TTL, err = deserializeTTL(data, offset)
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

func deserializeString(data []byte, offset int, fieldName string) (str string, newOffset int, err error) {
	strLen, n := binary.Uvarint(data[offset:])
	if n <= 0 {
		err = fmt.Errorf("invalid %s length", fieldName)
		return
	}
	if strLen > uint64(len(data)) || strLen > uint64(^uint(0)>>1) {
		err = fmt.Errorf("%s length too large", fieldName)
		return
	}
	offset += n

	strLenInt := int(strLen)
	if offset+strLenInt > len(data) {
		err = fmt.Errorf("%s data truncated", fieldName)
		return
	}
	str = string(data[offset : offset+strLenInt])
	newOffset = offset + strLenInt
	return
}

func deserializeStringSlice(data []byte, offset int) (args []string, newOffset int, err error) {
	argsCount, n := binary.Uvarint(data[offset:])
	if n <= 0 {
		err = fmt.Errorf("invalid args count")
		return
	}
	offset += n

	args = make([]string, argsCount)
	for i := uint64(0); i < argsCount; i++ {
		var arg string
		arg, offset, err = deserializeString(data, offset, "arg")
		if err != nil {
			return
		}
		args[i] = arg
	}

	newOffset = offset
	return
}

func deserializeTTL(data []byte, offset int) (time.Duration, error) {
	ttlSeconds, n := binary.Uvarint(data[offset:])
	if n <= 0 {
		return 0, fmt.Errorf("invalid TTL")
	}
	if ttlSeconds > uint64(maxInt64Value) {
		return 0, fmt.Errorf("TTL too large")
	}
	return time.Duration(int64(ttlSeconds)) * time.Second, nil
}

// Serialize converts a Response into its binary representation for network transmission.
// The format varies by response type:
//   - RespOK/RespNil: just the type byte
//   - RespError/RespString: type + varint length + data bytes
//   - RespInt: type + varint-encoded signed integer
//   - RespArray: type + varint count + (varint length + bytes) for each item
//
// Example:
//
//	resp := &Response{Type: RespString, Data: "hello"}
//	data, err := resp.Serialize()
//
// Returns:
//   - Binary representation of the response
//   - Error if serialization fails
func (r *Response) Serialize() ([]byte, error) {
	var buf []byte

	buf = append(buf, byte(r.Type))

	switch r.Type {
	case RespOK:
		return buf, nil
	case RespError:
		errorBytes := []byte(r.Error)
		buf = binary.AppendUvarint(buf, uint64(len(errorBytes)))
		buf = append(buf, errorBytes...)
	case RespString:
		if str, ok := r.Data.(string); ok {
			strBytes := []byte(str)
			buf = binary.AppendUvarint(buf, uint64(len(strBytes)))
			buf = append(buf, strBytes...)
		}
	case RespInt:
		if num, ok := r.Data.(int64); ok {
			buf = binary.AppendVarint(buf, num)
		}
	case RespArray:
		if arr, ok := r.Data.([]string); ok {
			buf = binary.AppendUvarint(buf, uint64(len(arr)))
			for _, item := range arr {
				itemBytes := []byte(item)
				buf = binary.AppendUvarint(buf, uint64(len(itemBytes)))
				buf = append(buf, itemBytes...)
			}
		}
	case RespNil:
		return buf, nil
	}

	return buf, nil
}

// DeserializeResponse reconstructs a Response from its binary representation.
// This is the inverse operation of Response.Serialize().
//
// Parameters:
//   - data: Binary data containing the serialized response
//
// Returns:
//   - Reconstructed Response object
//   - Error if deserialization fails or data is corrupted
//
// Example:
//
//	resp, err := protocol.DeserializeResponse(data)
//	if err != nil {
//		log.Printf("Failed to deserialize response: %v", err)
//		return
//	}
func DeserializeResponse(data []byte) (*Response, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty response data")
	}

	resp := &Response{}
	offset := 0

	resp.Type = ResponseType(data[offset])
	offset++

	switch resp.Type {
	case RespOK, RespNil:
		return resp, nil
	case RespError:
		return deserializeErrorResponse(resp, data, offset)
	case RespString:
		return deserializeStringResponse(resp, data, offset)
	case RespInt:
		return deserializeIntResponse(resp, data, offset)
	case RespArray:
		return deserializeArrayResponse(resp, data, offset)
	}

	return resp, nil
}

func deserializeErrorResponse(resp *Response, data []byte, offset int) (*Response, error) {
	errorStr, _, err := deserializeString(data, offset, "error")
	if err != nil {
		return nil, err
	}
	resp.Error = errorStr
	return resp, nil
}

func deserializeStringResponse(resp *Response, data []byte, offset int) (*Response, error) {
	str, _, err := deserializeString(data, offset, "string")
	if err != nil {
		return nil, err
	}
	resp.Data = str
	return resp, nil
}

func deserializeIntResponse(resp *Response, data []byte, offset int) (*Response, error) {
	num, n := binary.Varint(data[offset:])
	if n <= 0 {
		return nil, fmt.Errorf("invalid integer")
	}
	resp.Data = num
	return resp, nil
}

func deserializeArrayResponse(resp *Response, data []byte, offset int) (*Response, error) {
	arr, _, err := deserializeStringSlice(data, offset)
	if err != nil {
		return nil, err
	}
	resp.Data = arr
	return resp, nil
}

// ParseTextCommand parses a Redis-style text command into a Command struct.
// This is useful for debugging, testing, or implementing a text-based interface.
// Supports basic commands like GET, SET, DEL, EXISTS, INCR, DECR, PING.
//
// Example:
//
//	cmd, err := protocol.ParseTextCommand("SET mykey myvalue 60")
//	if err != nil {
//		log.Fatal(err)
//	}
//	// cmd.Type == CmdSet, cmd.Key == "mykey", cmd.Args == ["myvalue"], cmd.TTL == 60s
//
// Parameters:
//   - line: Text command in Redis format (space-separated)
//
// Returns:
//   - Parsed Command object
//   - Error if command is invalid or unsupported
func ParseTextCommand(line string) (*Command, error) {
	parts := strings.Fields(strings.TrimSpace(line))
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmdStr := strings.ToUpper(parts[0])

	switch cmdStr {
	case "GET":
		return parseGetCommand(parts)
	case "SET":
		return parseSetCommand(parts)
	case "DEL":
		return parseDelCommand(parts)
	case "EXISTS":
		return parseExistsCommand(parts)
	case "INCR":
		return parseIncrCommand(parts)
	case "DECR":
		return parseDecrCommand(parts)
	case "PING":
		return parsePingCommand()
	default:
		return nil, fmt.Errorf("unknown command: %s", cmdStr)
	}
}

func parseGetCommand(parts []string) (*Command, error) {
	if len(parts) != exactArgsForGet {
		return nil, fmt.Errorf("GET requires exactly 1 argument")
	}
	return &Command{Type: CmdGet, Key: parts[1]}, nil
}

func parseSetCommand(parts []string) (*Command, error) {
	if len(parts) < minArgsForSet {
		return nil, fmt.Errorf("SET requires at least 2 arguments")
	}

	cmd := &Command{
		Type: CmdSet,
		Key:  parts[1],
		Args: []string{parts[2]},
	}

	const maxArgsForSet = 4
	if len(parts) > maxArgsForSet {
		if ttl, err := strconv.Atoi(parts[3]); err == nil {
			cmd.TTL = time.Duration(ttl) * time.Second
		}
	}

	return cmd, nil
}

func parseDelCommand(parts []string) (*Command, error) {
	if len(parts) != exactArgsForDel {
		return nil, fmt.Errorf("DEL requires exactly 1 argument")
	}
	return &Command{Type: CmdDel, Key: parts[1]}, nil
}

func parseExistsCommand(parts []string) (*Command, error) {
	if len(parts) != exactArgsForExists {
		return nil, fmt.Errorf("EXISTS requires exactly 1 argument")
	}
	return &Command{Type: CmdExists, Key: parts[1]}, nil
}

func parseIncrCommand(parts []string) (*Command, error) {
	if len(parts) != exactArgsForIncr {
		return nil, fmt.Errorf("INCR requires exactly 1 argument")
	}
	return &Command{Type: CmdIncr, Key: parts[1]}, nil
}

func parseDecrCommand(parts []string) (*Command, error) {
	if len(parts) != exactArgsForDecr {
		return nil, fmt.Errorf("DECR requires exactly 1 argument")
	}
	return &Command{Type: CmdDecr, Key: parts[1]}, nil
}

func parsePingCommand() (*Command, error) {
	return &Command{Type: CmdPing}, nil
}

// WriteResponse writes a Response to the given writer with proper framing.
// The response is serialized and prefixed with a 4-byte length header.
// This ensures the receiver can read the complete message.
//
// Example:
//
//	resp := &Response{Type: RespString, Data: "hello"}
//	err := protocol.WriteResponse(conn, resp)
//
// Parameters:
//   - w: Writer to send the response to (typically a network connection)
//   - resp: Response object to send
//
// Returns:
//   - Error if writing fails
func WriteResponse(w io.Writer, resp *Response) error {
	data, err := resp.Serialize()
	if err != nil {
		return err
	}

	length := make([]byte, protocolHeaderSize)
	dataLen := len(data)
	if dataLen > maxUint32Value { // Check for uint32 overflow (max uint32)
		return fmt.Errorf("data too large")
	}
	binary.BigEndian.PutUint32(length, uint32(dataLen))

	if _, writeErr := w.Write(length); writeErr != nil {
		return writeErr
	}

	_, err = w.Write(data)
	return err
}

// ReadResponse reads a Response from the given reader.
// It first reads the 4-byte length header, then reads and deserializes
// the response data. Includes protection against oversized messages.
//
// Example:
//
//	resp, err := protocol.ReadResponse(conn)
//	if err != nil {
//		log.Printf("Failed to read response: %v", err)
//		return
//	}
//
// Parameters:
//   - r: Reader to read from (typically a network connection)
//
// Returns:
//   - Deserialized Response object
//   - Error if reading or deserialization fails
func ReadResponse(r io.Reader) (*Response, error) {
	lengthBuf := make([]byte, protocolHeaderSize)
	if _, err := io.ReadFull(r, lengthBuf); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length > 1024*1024 {
		return nil, fmt.Errorf("response too large: %d bytes", length)
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}

	return DeserializeResponse(data)
}

// WriteCommand writes a Command to the given writer with proper framing.
// The command is serialized and prefixed with a 4-byte length header.
//
// Example:
//
//	cmd := &Command{Type: CmdGet, Key: "mykey"}
//	err := protocol.WriteCommand(conn, cmd)
//
// Parameters:
//   - w: Writer to send the command to (typically a network connection)
//   - cmd: Command object to send
//
// Returns:
//   - Error if writing fails
func WriteCommand(w io.Writer, cmd *Command) error {
	data, err := cmd.Serialize()
	if err != nil {
		return err
	}

	length := make([]byte, protocolHeaderSize)
	dataLen := len(data)
	if dataLen > maxUint32Value { // Check for uint32 overflow (max uint32)
		return fmt.Errorf("data too large")
	}
	binary.BigEndian.PutUint32(length, uint32(dataLen))

	if _, writeErr := w.Write(length); writeErr != nil {
		return writeErr
	}

	_, err = w.Write(data)
	return err
}

// ReadCommand reads a Command from the given reader.
// It first reads the 4-byte length header, then reads and deserializes
// the command data. Includes protection against oversized messages.
//
// Example:
//
//	cmd, err := protocol.ReadCommand(conn)
//	if err != nil {
//		log.Printf("Failed to read command: %v", err)
//		return
//	}
//
// Parameters:
//   - r: Reader to read from (typically a network connection)
//
// Returns:
//   - Deserialized Command object
//   - Error if reading or deserialization fails
func ReadCommand(r io.Reader) (*Command, error) {
	lengthBuf := make([]byte, protocolHeaderSize)
	if _, err := io.ReadFull(r, lengthBuf); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuf)
	if length > 1024*1024 {
		return nil, fmt.Errorf("command too large: %d bytes", length)
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}

	return DeserializeCommand(data)
}
