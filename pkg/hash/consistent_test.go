package hash
import "fmt"

import (
	"testing"
)

func TestConsistentHash(t *testing.T) {
	ch := New(3)
	
	nodes := []string{"node1:8080", "node2:8080", "node3:8080"}
	for _, node := range nodes {
		ch.AddNode(node)
	}
	
	if len(ch.GetNodes()) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(ch.GetNodes()))
	}
	
	key1 := "test_key_1"
	key2 := "test_key_2"
	
	node1 := ch.GetNode(key1)
	node2 := ch.GetNode(key2)
	
	if node1 == "" || node2 == "" {
		t.Error("GetNode returned empty string")
	}
	
	for i := 0; i < 10; i++ {
		if ch.GetNode(key1) != node1 {
			t.Error("GetNode should be consistent")
		}
	}
	
	ch.RemoveNode("node1:8080")
	if len(ch.GetNodes()) != 2 {
		t.Errorf("Expected 2 nodes after removal, got %d", len(ch.GetNodes()))
	}
	
	newNode1 := ch.GetNode(key1)
	if newNode1 == "node1:8080" {
		t.Error("Removed node should not be returned")
	}
}

func TestConsistentHashDistribution(t *testing.T) {
	ch := New(150)
	
	nodes := []string{"node1:8080", "node2:8080", "node3:8080"}
	for _, node := range nodes {
		ch.AddNode(node)
	}
	
	distribution := make(map[string]int)
	
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key_%d", i)
		node := ch.GetNode(key)
		distribution[node]++
	}
	
	for node, count := range distribution {
		if count < 200 || count > 500 {
			t.Errorf("Poor distribution for node %s: %d keys", node, count)
		}
	}
}
