package hash

import (
	"fmt"
	"testing"
)

func TestConsistentHash(t *testing.T) {
	ch := New(3)

	nodes := []string{"node1:8080", "node2:8080", "node3:8080"}
	for _, node := range nodes {
		ch.AddNode(node)
	}

	if got := len(ch.GetNodes()); got != 3 {
		t.Errorf("expected 3 nodes, got %d", got)
	}

	key1 := "test_key_1"
	key2 := "test_key_2"

	node1 := ch.GetNode(key1)
	node2 := ch.GetNode(key2)

	if node1 == "" || node2 == "" {
		t.Error("GetNode returned empty string for non-empty ring")
	}

	for range 10 {
		if ch.GetNode(key1) != node1 {
			t.Error("GetNode should be deterministic")
		}
	}

	ch.RemoveNode("node1:8080")
	if got := len(ch.GetNodes()); got != 2 {
		t.Errorf("expected 2 nodes after removal, got %d", got)
	}

	if ch.GetNode(key1) == "node1:8080" {
		t.Error("removed node should not be returned")
	}
}

func TestConsistentHashEmptyRing(t *testing.T) {
	ch := New(10)
	if node := ch.GetNode("any"); node != "" {
		t.Errorf("expected empty string from empty ring, got %q", node)
	}
}

func TestConsistentHashDuplicateAdd(t *testing.T) {
	ch := New(10)
	ch.AddNode("node:8080")
	ch.AddNode("node:8080") // duplicate
	if got := len(ch.GetNodes()); got != 1 {
		t.Errorf("expected 1 node after duplicate add, got %d", got)
	}
}

func TestConsistentHashRemoveNonExistent(t *testing.T) {
	ch := New(10)
	ch.AddNode("node:8080")
	ch.RemoveNode("other:8080") // should be a no-op
	if got := len(ch.GetNodes()); got != 1 {
		t.Errorf("expected 1 node after removing non-existent, got %d", got)
	}
}

func TestConsistentHashDistribution(t *testing.T) {
	ch := New(150)

	nodes := []string{"node1:8080", "node2:8080", "node3:8080"}
	for _, node := range nodes {
		ch.AddNode(node)
	}

	distribution := make(map[string]int)
	const total = 1000
	for i := range total {
		key := fmt.Sprintf("key_%d", i)
		distribution[ch.GetNode(key)]++
	}

	for node, count := range distribution {
		// With 150 virtual nodes each of 3 nodes, distribution should be rough thirds.
		if count < 200 || count > 500 {
			t.Errorf("poor distribution for node %s: %d/%d keys", node, count, total)
		}
	}
}

func TestConsistentHashStats(t *testing.T) {
	ch := New(10)
	ch.AddNode("n1:8080")
	ch.AddNode("n2:8080")

	stats := ch.Stats()
	if nodes, _ := stats["nodes"].(int); nodes != 2 {
		t.Errorf("expected 2 nodes in stats, got %d", nodes)
	}
	if vn, _ := stats["virtual_nodes"].(int); vn != 20 {
		t.Errorf("expected 20 virtual nodes, got %d", vn)
	}
}

// --- Benchmarks ---

func BenchmarkGetNode(b *testing.B) {
	ch := New(150)
	ch.AddNode("node1:8080")
	ch.AddNode("node2:8080")
	ch.AddNode("node3:8080")
	var i int
	for b.Loop() {
		ch.GetNode(fmt.Sprintf("key:%d", i))
		i++
	}
}

func BenchmarkAddNode(b *testing.B) {
	var i int
	for b.Loop() {
		ch := New(150)
		ch.AddNode(fmt.Sprintf("node%d:8080", i))
		i++
	}
}
