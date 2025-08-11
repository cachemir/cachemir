package cache

import (
	"testing"
	"time"
)

func TestCacheBasicOperations(t *testing.T) {
	c := New()
	
	c.Set("key1", "value1", 0)
	
	if value, exists := c.Get("key1"); !exists || value != "value1" {
		t.Errorf("Expected value1, got %s (exists: %t)", value, exists)
	}
	
	if !c.Exists("key1") {
		t.Error("Key should exist")
	}
	
	if !c.Del("key1") {
		t.Error("Delete should return true")
	}
	
	if c.Exists("key1") {
		t.Error("Key should not exist after deletion")
	}
}

func TestCacheExpiration(t *testing.T) {
	c := New()
	
	c.Set("temp_key", "temp_value", 100*time.Millisecond)
	
	if value, exists := c.Get("temp_key"); !exists || value != "temp_value" {
		t.Errorf("Expected temp_value, got %s (exists: %t)", value, exists)
	}
	
	time.Sleep(150 * time.Millisecond)
	
	if value, exists := c.Get("temp_key"); exists {
		t.Errorf("Key should have expired, but got %s", value)
	}
}

func TestCacheIncrement(t *testing.T) {
	c := New()
	
	value, err := c.Incr("counter")
	if err != nil || value != 1 {
		t.Errorf("Expected 1, got %d (error: %v)", value, err)
	}
	
	value, err = c.Incr("counter")
	if err != nil || value != 2 {
		t.Errorf("Expected 2, got %d (error: %v)", value, err)
	}
	
	value, err = c.Decr("counter")
	if err != nil || value != 1 {
		t.Errorf("Expected 1, got %d (error: %v)", value, err)
	}
}

func TestCacheHashOperations(t *testing.T) {
	c := New()
	
	c.HSet("hash1", "field1", "value1")
	c.HSet("hash1", "field2", "value2")
	
	if value, exists := c.HGet("hash1", "field1"); !exists || value != "value1" {
		t.Errorf("Expected value1, got %s (exists: %t)", value, exists)
	}
	
	hash := c.HGetAll("hash1")
	if len(hash) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(hash))
	}
	
	if hash["field1"] != "value1" || hash["field2"] != "value2" {
		t.Errorf("Hash values incorrect: %+v", hash)
	}
	
	if !c.HDel("hash1", "field1") {
		t.Error("HDel should return true")
	}
	
	if value, exists := c.HGet("hash1", "field1"); exists {
		t.Errorf("Field should not exist after deletion, got %s", value)
	}
}

func TestCacheListOperations(t *testing.T) {
	c := New()
	
	length := c.LPush("list1", "item1", "item1")
	if length != 2 {
		t.Errorf("Expected length 2, got %d", length)
	}
	
	length = c.RPush("list1", "item3", "item4")
	if length != 4 {
		t.Errorf("Expected length 4, got %d", length)
	}
	
	if value, exists := c.LPop("list1"); !exists || value != "item1" {
		t.Errorf("Expected item1, got %s (exists: %t)", value, exists)
	}
	
	if value, exists := c.RPop("list1"); !exists || value != "item4" {
		t.Errorf("Expected item4, got %s (exists: %t)", value, exists)
	}
	
	if length := c.LLen("list1"); length != 2 {
		t.Errorf("Expected length 2, got %d", length)
	}
}

func TestCacheSetOperations(t *testing.T) {
	c := New()
	
	added := c.SAdd("set1", "member1", "member2", "member3")
	if added != 3 {
		t.Errorf("Expected 3 added, got %d", added)
	}
	
	added = c.SAdd("set1", "member2", "member4")
	if added != 1 {
		t.Errorf("Expected 1 added (member4), got %d", added)
	}
	
	if !c.SIsMember("set1", "member1") {
		t.Error("member1 should be in set")
	}
	
	if c.SIsMember("set1", "nonexistent") {
		t.Error("nonexistent should not be in set")
	}
	
	members := c.SMembers("set1")
	if len(members) != 4 {
		t.Errorf("Expected 4 members, got %d", len(members))
	}
	
	removed := c.SRem("set1", "member1", "member2")
	if removed != 2 {
		t.Errorf("Expected 2 removed, got %d", removed)
	}
}
