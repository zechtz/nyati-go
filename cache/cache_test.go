package cache

import (
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	if cache == nil {
		t.Error("NewCache should not return nil")
	}
	
	if cache.ttl != 5*time.Minute {
		t.Errorf("Cache TTL = %v, want 5m", cache.ttl)
	}
	
	if cache.Size() != 0 {
		t.Error("New cache should be empty")
	}
}

func TestCacheSetAndGet(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	
	// Test setting and getting a value
	cache.Set("key1", "value1")
	
	value, exists := cache.Get("key1")
	if !exists {
		t.Error("Value should exist in cache")
	}
	
	if value != "value1" {
		t.Errorf("Got value %v, want value1", value)
	}
	
	// Test getting non-existent key
	_, exists = cache.Get("nonexistent")
	if exists {
		t.Error("Non-existent key should not exist")
	}
}

func TestCacheSetWithTTL(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	
	// Set with very short TTL
	cache.SetWithTTL("short_ttl", "value", 10*time.Millisecond)
	
	// Should exist immediately
	value, exists := cache.Get("short_ttl")
	if !exists || value != "value" {
		t.Error("Value should exist immediately after setting")
	}
	
	// Wait for expiration
	time.Sleep(20 * time.Millisecond)
	
	// Should not exist after expiration
	_, exists = cache.Get("short_ttl")
	if exists {
		t.Error("Value should not exist after TTL expiration")
	}
}

func TestCacheDelete(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	
	if cache.Size() != 2 {
		t.Error("Cache should contain 2 items")
	}
	
	cache.Delete("key1")
	
	if cache.Size() != 1 {
		t.Error("Cache should contain 1 item after deletion")
	}
	
	_, exists := cache.Get("key1")
	if exists {
		t.Error("Deleted key should not exist")
	}
	
	_, exists = cache.Get("key2")
	if !exists {
		t.Error("Non-deleted key should still exist")
	}
}

func TestCacheClear(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")
	
	if cache.Size() != 3 {
		t.Error("Cache should contain 3 items")
	}
	
	cache.Clear()
	
	if cache.Size() != 0 {
		t.Error("Cache should be empty after clear")
	}
}

func TestCacheKeys(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")
	
	keys := cache.Keys()
	
	if len(keys) != 3 {
		t.Errorf("Should have 3 keys, got %d", len(keys))
	}
	
	// Check that all expected keys are present
	keyMap := make(map[string]bool)
	for _, key := range keys {
		keyMap[key] = true
	}
	
	expectedKeys := []string{"key1", "key2", "key3"}
	for _, expectedKey := range expectedKeys {
		if !keyMap[expectedKey] {
			t.Errorf("Expected key %s not found in keys", expectedKey)
		}
	}
}

func TestCacheStats(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	
	// Add some items
	cache.Set("key1", "value1")
	cache.SetWithTTL("key2", "value2", 10*time.Millisecond)
	cache.Set("key3", "value3")
	
	// Get stats before expiration
	stats := cache.Stats()
	
	if stats["total_items"] != 3 {
		t.Errorf("Total items = %v, want 3", stats["total_items"])
	}
	
	if stats["default_ttl"] != (5 * time.Minute).String() {
		t.Errorf("Default TTL = %v, want %s", stats["default_ttl"], (5*time.Minute).String())
	}
	
	// Wait for one item to expire
	time.Sleep(20 * time.Millisecond)
	
	// Get stats after expiration
	stats = cache.Stats()
	
	// Note: The expired item might still be in the cache until cleanup
	if stats["total_items"].(int) < 2 || stats["total_items"].(int) > 3 {
		t.Errorf("Total items after expiration = %v, should be 2 or 3", stats["total_items"])
	}
}

func TestCacheGetOrSet(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	
	callCount := 0
	valueFunc := func() interface{} {
		callCount++
		return "generated_value"
	}
	
	// First call should generate the value
	value := cache.GetOrSet("key1", valueFunc)
	if value != "generated_value" {
		t.Errorf("Got value %v, want generated_value", value)
	}
	if callCount != 1 {
		t.Errorf("Value function called %d times, want 1", callCount)
	}
	
	// Second call should use cached value
	value = cache.GetOrSet("key1", valueFunc)
	if value != "generated_value" {
		t.Errorf("Got value %v, want generated_value", value)
	}
	if callCount != 1 {
		t.Errorf("Value function called %d times, want 1 (should use cache)", callCount)
	}
}

func TestCacheGetOrSetWithTTL(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	
	callCount := 0
	valueFunc := func() interface{} {
		callCount++
		return "generated_value"
	}
	
	// Set with very short TTL
	value := cache.GetOrSetWithTTL("key1", valueFunc, 10*time.Millisecond)
	if value != "generated_value" {
		t.Errorf("Got value %v, want generated_value", value)
	}
	if callCount != 1 {
		t.Errorf("Value function called %d times, want 1", callCount)
	}
	
	// Wait for expiration
	time.Sleep(20 * time.Millisecond)
	
	// Should call function again after expiration
	value = cache.GetOrSetWithTTL("key1", valueFunc, 10*time.Millisecond)
	if value != "generated_value" {
		t.Errorf("Got value %v, want generated_value", value)
	}
	if callCount != 2 {
		t.Errorf("Value function called %d times, want 2 (after expiration)", callCount)
	}
}

func TestCacheMemoryStats(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	
	stats := cache.MemoryStats()
	
	if _, exists := stats["estimated_memory_bytes"]; !exists {
		t.Error("Memory stats should include estimated_memory_bytes")
	}
	
	if stats["item_count"] != 2 {
		t.Errorf("Item count = %v, want 2", stats["item_count"])
	}
}

func TestCacheItemIsExpired(t *testing.T) {
	// Test expired item
	expiredItem := &CacheItem{
		Value:     "test",
		ExpiresAt: time.Now().Add(-1 * time.Minute),
	}
	
	if !expiredItem.IsExpired() {
		t.Error("Item with past expiration should be expired")
	}
	
	// Test non-expired item
	activeItem := &CacheItem{
		Value:     "test",
		ExpiresAt: time.Now().Add(1 * time.Minute),
	}
	
	if activeItem.IsExpired() {
		t.Error("Item with future expiration should not be expired")
	}
}

func TestCacheConcurrency(t *testing.T) {
	cache := NewCache(5 * time.Minute)
	
	// Test concurrent access
	done := make(chan bool, 2)
	
	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cache.Set("key", i)
		}
		done <- true
	}()
	
	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cache.Get("key")
		}
		done <- true
	}()
	
	// Wait for both goroutines to complete
	<-done
	<-done
	
	// Should not panic and cache should still be functional
	cache.Set("final_key", "final_value")
	value, exists := cache.Get("final_key")
	if !exists || value != "final_value" {
		t.Error("Cache should still be functional after concurrent access")
	}
}