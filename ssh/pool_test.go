package ssh

import (
	"testing"
	"time"

	"github.com/zechtz/nyatictl/config"
)

func TestNewConnectionPool(t *testing.T) {
	// Test with nil config (should use defaults)
	pool := NewConnectionPool(nil)
	if pool == nil {
		t.Error("NewConnectionPool() should not return nil")
	}
	defer pool.Close()

	stats := pool.Stats()
	if stats["total_connections"] != 0 {
		t.Error("New pool should have 0 connections")
	}
	if stats["max_idle"] != 5 {
		t.Error("Default max_idle should be 5")
	}

	// Test with custom config
	customConfig := &ConnectionPoolConfig{
		MaxIdle:     10,
		MaxLifetime: 10 * time.Minute,
		IdleTimeout: 2 * time.Minute,
	}
	customPool := NewConnectionPool(customConfig)
	defer customPool.Close()

	customStats := customPool.Stats()
	if customStats["max_idle"] != 10 {
		t.Error("Custom max_idle should be 10")
	}
}

func TestConnectionPoolStats(t *testing.T) {
	pool := NewConnectionPool(nil)
	defer pool.Close()

	stats := pool.Stats()
	
	expectedKeys := []string{"total_connections", "in_use", "idle", "max_idle", "max_lifetime", "idle_timeout"}
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Stats should contain key: %s", key)
		}
	}

	if stats["total_connections"] != 0 {
		t.Error("Initial total_connections should be 0")
	}
	if stats["in_use"] != 0 {
		t.Error("Initial in_use should be 0")
	}
	if stats["idle"] != 0 {
		t.Error("Initial idle should be 0")
	}
}

func TestManagerPoolingFunctions(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]config.Host{
			"testhost": {
				Host:     "example.com",
				Username: "user",
				Password: "pass",
			},
		},
	}

	manager, err := NewManager(cfg, []string{"testhost"}, false)
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}
	defer manager.Close()

	// Test initial state
	stats := manager.GetPoolStats()
	if stats["pooling_enabled"] != false {
		t.Error("Pooling should be disabled by default")
	}

	// Test enabling pooling
	manager.EnableConnectionPooling(nil)
	stats = manager.GetPoolStats()
	if stats["pooling_enabled"] != true {
		t.Error("Pooling should be enabled after EnableConnectionPooling()")
	}

	// Test disabling pooling
	manager.DisableConnectionPooling()
	stats = manager.GetPoolStats()
	if stats["pooling_enabled"] != false {
		t.Error("Pooling should be disabled after DisableConnectionPooling()")
	}
}

func TestDefaultPoolConfig(t *testing.T) {
	cfg := defaultPoolConfig()
	
	if cfg.MaxIdle != 5 {
		t.Errorf("Default MaxIdle = %d, want 5", cfg.MaxIdle)
	}
	if cfg.MaxLifetime != 5*time.Minute {
		t.Errorf("Default MaxLifetime = %v, want 5m", cfg.MaxLifetime)
	}
	if cfg.IdleTimeout != 1*time.Minute {
		t.Errorf("Default IdleTimeout = %v, want 1m", cfg.IdleTimeout)
	}
}

func TestPooledConnectionUsability(t *testing.T) {
	// Test with nil client
	conn := &PooledConnection{
		client:    nil,
		host:      "test",
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		inUse:     false,
	}

	if conn.isUsable() {
		t.Error("Connection with nil client should not be usable")
	}
}

func TestConnectionPoolCleanup(t *testing.T) {
	// Create pool with very short timeouts for testing
	cfg := &ConnectionPoolConfig{
		MaxIdle:     1,
		MaxLifetime: 100 * time.Millisecond,
		IdleTimeout: 50 * time.Millisecond,
	}
	
	pool := NewConnectionPool(cfg)
	defer pool.Close()

	// Add a mock connection directly to the pool for testing cleanup
	mockConn := &PooledConnection{
		client:    nil, // This will make it unusable
		host:      "test@example.com",
		createdAt: time.Now().Add(-200 * time.Millisecond), // Old connection
		lastUsed:  time.Now().Add(-100 * time.Millisecond), // Idle connection
		inUse:     false,
	}

	pool.poolLock.Lock()
	pool.pool["test@example.com"] = mockConn
	pool.poolLock.Unlock()

	// Trigger cleanup
	pool.cleanup()

	// Check that the expired connection was removed
	pool.poolLock.RLock()
	_, exists := pool.pool["test@example.com"]
	pool.poolLock.RUnlock()

	if exists {
		t.Error("Expired connection should have been removed by cleanup")
	}
}

func TestReleaseConnection(t *testing.T) {
	pool := NewConnectionPool(nil)
	defer pool.Close()

	// Test releasing nil connection (should not panic)
	pool.ReleaseConnection(nil)

	// Test releasing valid connection
	conn := &PooledConnection{
		client:    nil,
		host:      "test",
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		inUse:     true,
	}

	pool.ReleaseConnection(conn)
	
	if conn.inUse {
		t.Error("Connection should not be in use after release")
	}
}