package cache

import (
	"sync"
	"time"
)

// CacheItem represents a cached item with expiration
type CacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

// IsExpired returns true if the cache item has expired
func (item *CacheItem) IsExpired() bool {
	return time.Now().After(item.ExpiresAt)
}

// Cache represents an in-memory cache with TTL support
type Cache struct {
	items map[string]*CacheItem
	mutex sync.RWMutex
	ttl   time.Duration
}

// NewCache creates a new cache with the specified default TTL
func NewCache(ttl time.Duration) *Cache {
	cache := &Cache{
		items: make(map[string]*CacheItem),
		ttl:   ttl,
	}
	
	// Start cleanup goroutine
	go cache.cleanupExpired()
	
	return cache
}

// Set stores a value in the cache with the default TTL
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value in the cache with a custom TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.items[key] = &CacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	item, exists := c.items[key]
	if !exists {
		return nil, false
	}
	
	if item.IsExpired() {
		// Remove expired item
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.items, key)
		c.mutex.Unlock()
		c.mutex.RLock()
		return nil, false
	}
	
	return item.Value, true
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.items = make(map[string]*CacheItem)
}

// Size returns the number of items in the cache
func (c *Cache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return len(c.items)
}

// Keys returns all keys in the cache
func (c *Cache) Keys() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	keys := make([]string, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}
	
	return keys
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	expired := 0
	active := 0
	now := time.Now()
	
	for _, item := range c.items {
		if now.After(item.ExpiresAt) {
			expired++
		} else {
			active++
		}
	}
	
	return map[string]interface{}{
		"total_items":  len(c.items),
		"active_items": active,
		"expired_items": expired,
		"default_ttl":  c.ttl.String(),
	}
}

// cleanupExpired periodically removes expired items
func (c *Cache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		toDelete := make([]string, 0)
		
		for key, item := range c.items {
			if now.After(item.ExpiresAt) {
				toDelete = append(toDelete, key)
			}
		}
		
		for _, key := range toDelete {
			delete(c.items, key)
		}
		c.mutex.Unlock()
	}
}

// GetOrSet retrieves a value from the cache, or sets and returns it if not found
func (c *Cache) GetOrSet(key string, valueFunc func() interface{}) interface{} {
	return c.GetOrSetWithTTL(key, valueFunc, c.ttl)
}

// GetOrSetWithTTL retrieves a value from the cache, or sets and returns it with custom TTL if not found
func (c *Cache) GetOrSetWithTTL(key string, valueFunc func() interface{}, ttl time.Duration) interface{} {
	// First try to get from cache
	if value, exists := c.Get(key); exists {
		return value
	}
	
	// Generate the value
	value := valueFunc()
	
	// Store in cache
	c.SetWithTTL(key, value, ttl)
	
	return value
}

// MemoryStats returns memory usage statistics for the cache
func (c *Cache) MemoryStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	// This is a simplified memory calculation
	// In a production system, you might want more sophisticated memory tracking
	estimatedMemory := len(c.items) * 64 // Rough estimate per item
	
	return map[string]interface{}{
		"estimated_memory_bytes": estimatedMemory,
		"item_count":            len(c.items),
	}
}