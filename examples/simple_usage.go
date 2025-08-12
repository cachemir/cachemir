package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cachemir/cachemir/pkg/client"
)

const (
	tempKeyTTL     = 5 * time.Second
	expirationWait = 6 * time.Second
)

func main() {
	// Connect to CacheMir cluster
	c := client.New([]string{"localhost:8080"})
	defer func() {
		if err := c.Close(); err != nil {
			log.Printf("Error closing client: %v", err)
		}
	}()

	demonstrateStringOperations(c)
	demonstrateExpirationFeatures(c)
	demonstrateNumericOperations(c)
	demonstrateHashOperations(c)
	demonstrateListOperations(c)
	demonstrateSetOperations(c)
}

func demonstrateStringOperations(c *client.Client) {
	fmt.Println("=== String Operations ===")

	// Set a key-value pair
	if err := c.Set("greeting", "Hello, CacheMir!", 0); err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// Get the value
	if value, err := c.Get("greeting"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("greeting: %s\n", value)
	}
}

func demonstrateExpirationFeatures(c *client.Client) {
	// Set with expiration
	if err := c.Set("temp", "This will expire", tempKeyTTL); err != nil {
		log.Printf("Error: %v", err)
	}

	// Check TTL
	if ttl, err := c.TTL("temp"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("TTL for temp: %v\n", ttl)
	}

	// Wait for expiration
	fmt.Println("Waiting for expiration...")
	time.Sleep(expirationWait)

	// Try to get expired key
	if _, err := c.Get("temp"); err != nil {
		fmt.Printf("Key expired: %v\n", err)
	}
}

func demonstrateNumericOperations(c *client.Client) {
	fmt.Println("\n=== Numeric Operations ===")

	// Increment operations
	if count, err := c.Incr("counter"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Counter: %d\n", count)
	}
}

func demonstrateHashOperations(c *client.Client) {
	fmt.Println("\n=== Hash Operations ===")

	// Set user profile
	if err := c.HSet("user:123", "name", "Alice"); err != nil {
		log.Printf("Error setting name: %v", err)
	}
	if err := c.HSet("user:123", "email", "alice@example.com"); err != nil {
		log.Printf("Error setting email: %v", err)
	}
	if err := c.HSet("user:123", "age", "30"); err != nil {
		log.Printf("Error setting age: %v", err)
	}

	// Get specific field
	if name, err := c.HGet("user:123", "name"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("User name: %s\n", name)
	}

	// Get all fields
	if profile, err := c.HGetAll("user:123"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("User profile: %+v\n", profile)
	}
}

func demonstrateListOperations(c *client.Client) {
	fmt.Println("\n=== List Operations ===")

	// Add items to shopping cart
	if _, err := c.LPush("cart:123", "laptop", "mouse", "keyboard"); err != nil {
		log.Printf("Error adding items to cart: %v", err)
	}
	if _, err := c.RPush("cart:123", "monitor", "speakers"); err != nil {
		log.Printf("Error adding items to cart: %v", err)
	}

	// Remove items
	if item, err := c.LPop("cart:123"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Removed from cart: %s\n", item)
	}
}

func demonstrateSetOperations(c *client.Client) {
	fmt.Println("\n=== Set Operations ===")

	// Add tags
	if _, err := c.SAdd("tags:article:1", "golang", "cache", "distributed"); err != nil {
		log.Printf("Error: %v", err)
	}

	// Get all tags
	if tags, err := c.SMembers("tags:article:1"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Article tags: %v\n", tags)
	}

	// Cleanup
	fmt.Println("\n=== Cleanup ===")
	keys := []string{"greeting", "counter", "user:123", "cart:123", "tags:article:1"}
	for _, key := range keys {
		if deleted, err := c.Del(key); err != nil {
			log.Printf("Error deleting %s: %v", key, err)
		} else if deleted {
			fmt.Printf("Deleted key: %s\n", key)
		}
	}
}
