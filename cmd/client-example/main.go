package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cachemir/cachemir/pkg/client"
)

func main() {
	nodes := []string{"localhost:8080", "localhost:8081", "localhost:8082"}
	
	c := client.New(nodes)
	defer c.Close()
	
	fmt.Println("=== CacheMir Client Example ===")
	
	if err := c.Ping(); err != nil {
		log.Printf("Warning: Ping failed: %v", err)
	} else {
		fmt.Println("✓ Connected to CacheMir cluster")
	}
	
	fmt.Println("\n--- String Operations ---")
	
	if err := c.Set("user:1", "john_doe", 0); err != nil {
		log.Printf("SET failed: %v", err)
	} else {
		fmt.Println("✓ SET user:1 = john_doe")
	}
	
	if value, err := c.Get("user:1"); err != nil {
		log.Printf("GET failed: %v", err)
	} else {
		fmt.Printf("✓ GET user:1 = %s\n", value)
	}
	
	if exists, err := c.Exists("user:1"); err != nil {
		log.Printf("EXISTS failed: %v", err)
	} else {
		fmt.Printf("✓ EXISTS user:1 = %t\n", exists)
	}
	
	fmt.Println("\n--- Counter Operations ---")
	
	if value, err := c.Incr("counter"); err != nil {
		log.Printf("INCR failed: %v", err)
	} else {
		fmt.Printf("✓ INCR counter = %d\n", value)
	}
	
	if value, err := c.Incr("counter"); err != nil {
		log.Printf("INCR failed: %v", err)
	} else {
		fmt.Printf("✓ INCR counter = %d\n", value)
	}
	
	if value, err := c.Decr("counter"); err != nil {
		log.Printf("DECR failed: %v", err)
	} else {
		fmt.Printf("✓ DECR counter = %d\n", value)
	}
	
	fmt.Println("\n--- Expiration ---")
	
	if err := c.Set("temp_key", "temp_value", 5*time.Second); err != nil {
		log.Printf("SET with TTL failed: %v", err)
	} else {
		fmt.Println("✓ SET temp_key with 5s TTL")
	}
	
	if ttl, err := c.TTL("temp_key"); err != nil {
		log.Printf("TTL failed: %v", err)
	} else {
		fmt.Printf("✓ TTL temp_key = %v\n", ttl)
	}
	
	fmt.Println("\n--- Hash Operations ---")
	
	if err := c.HSet("user:1:profile", "name", "John Doe"); err != nil {
		log.Printf("HSET failed: %v", err)
	} else {
		fmt.Println("✓ HSET user:1:profile name = John Doe")
	}
	
	if err := c.HSet("user:1:profile", "email", "john@example.com"); err != nil {
		log.Printf("HSET failed: %v", err)
	} else {
		fmt.Println("✓ HSET user:1:profile email = john@example.com")
	}
	
	if value, err := c.HGet("user:1:profile", "name"); err != nil {
		log.Printf("HGET failed: %v", err)
	} else {
		fmt.Printf("✓ HGET user:1:profile name = %s\n", value)
	}
	
	if hash, err := c.HGetAll("user:1:profile"); err != nil {
		log.Printf("HGETALL failed: %v", err)
	} else {
		fmt.Printf("✓ HGETALL user:1:profile = %+v\n", hash)
	}
	
	fmt.Println("\n--- List Operations ---")
	
	if length, err := c.LPush("tasks", "task1", "task2", "task3"); err != nil {
		log.Printf("LPUSH failed: %v", err)
	} else {
		fmt.Printf("✓ LPUSH tasks = %d items\n", length)
	}
	
	if length, err := c.RPush("tasks", "task4", "task5"); err != nil {
		log.Printf("RPUSH failed: %v", err)
	} else {
		fmt.Printf("✓ RPUSH tasks = %d items\n", length)
	}
	
	if value, err := c.LPop("tasks"); err != nil {
		log.Printf("LPOP failed: %v", err)
	} else {
		fmt.Printf("✓ LPOP tasks = %s\n", value)
	}
	
	fmt.Println("\n--- Set Operations ---")
	
	if added, err := c.SAdd("tags", "go", "cache", "distributed", "redis"); err != nil {
		log.Printf("SADD failed: %v", err)
	} else {
		fmt.Printf("✓ SADD tags = %d members added\n", added)
	}
	
	if members, err := c.SMembers("tags"); err != nil {
		log.Printf("SMEMBERS failed: %v", err)
	} else {
		fmt.Printf("✓ SMEMBERS tags = %v\n", members)
	}
	
	fmt.Println("\n--- Cleanup ---")
	
	if deleted, err := c.Del("user:1"); err != nil {
		log.Printf("DEL failed: %v", err)
	} else {
		fmt.Printf("✓ DEL user:1 = %t\n", deleted)
	}
	
	fmt.Println("\n=== Example Complete ===")
}
