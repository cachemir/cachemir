package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cachemir/cachemir/pkg/client"
)

func main() {
	// Connect to CacheMir cluster
	client := client.New([]string{"localhost:8080"})
	defer client.Close()

	// Basic string operations
	fmt.Println("=== String Operations ===")
	
	// Set a key-value pair
	if err := client.Set("greeting", "Hello, CacheMir!", 0); err != nil {
		log.Fatal(err)
	}
	
	// Get the value
	if value, err := client.Get("greeting"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("greeting = %s\n", value)
	}
	
	// Set with expiration
	if err := client.Set("temp", "This will expire", 5*time.Second); err != nil {
		log.Fatal(err)
	}
	
	// Check TTL
	if ttl, err := client.TTL("temp"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("temp TTL = %v\n", ttl)
	}

	// Counter operations
	fmt.Println("\n=== Counter Operations ===")
	
	// Increment counter
	for i := 0; i < 5; i++ {
		if count, err := client.Incr("page_views"); err != nil {
			log.Printf("Error: %v", err)
		} else {
			fmt.Printf("Page views: %d\n", count)
		}
	}

	// Hash operations
	fmt.Println("\n=== Hash Operations ===")
	
	// Set user profile
	client.HSet("user:123", "name", "Alice")
	client.HSet("user:123", "email", "alice@example.com")
	client.HSet("user:123", "age", "30")
	
	// Get specific field
	if name, err := client.HGet("user:123", "name"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("User name: %s\n", name)
	}
	
	// Get all fields
	if profile, err := client.HGetAll("user:123"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("User profile: %+v\n", profile)
	}

	// List operations
	fmt.Println("\n=== List Operations ===")
	
	// Add items to shopping cart
	client.LPush("cart:123", "laptop", "mouse", "keyboard")
	client.RPush("cart:123", "monitor", "speakers")
	
	// Remove items
	if item, err := client.LPop("cart:123"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Removed from cart: %s\n", item)
	}

	// Set operations
	fmt.Println("\n=== Set Operations ===")
	
	// Add tags
	if added, err := client.SAdd("article:tags", "golang", "cache", "distributed", "performance"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Added %d tags\n", added)
	}
	
	// Get all tags
	if tags, err := client.SMembers("article:tags"); err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Article tags: %v\n", tags)
	}

	fmt.Println("\n=== Cleanup ===")
	
	// Delete keys
	keys := []string{"greeting", "temp", "page_views", "user:123", "cart:123", "article:tags"}
	for _, key := range keys {
		if deleted, err := client.Del(key); err != nil {
			log.Printf("Error deleting %s: %v", key, err)
		} else if deleted {
			fmt.Printf("Deleted: %s\n", key)
		}
	}
}
