package ssh

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zechtz/nyatictl/config"
	"github.com/zechtz/nyatictl/logger"
	"golang.org/x/crypto/ssh"
)

// ConnectionPool manages a pool of SSH connections for reuse
type ConnectionPool struct {
	pool        map[string]*PooledConnection // Pool of connections keyed by host identifier
	poolLock    sync.RWMutex                 // Protects the pool map
	maxIdle     int                          // Maximum number of idle connections per host
	maxLifetime time.Duration               // Maximum lifetime of a connection
	idleTimeout time.Duration               // Timeout for idle connections
}

// PooledConnection represents a connection in the pool with metadata
type PooledConnection struct {
	client      *ssh.Client
	host        string
	createdAt   time.Time
	lastUsed    time.Time
	inUse       bool
	useLock     sync.Mutex
}

// ConnectionPoolConfig holds configuration for the connection pool
type ConnectionPoolConfig struct {
	MaxIdle     int           `default:"5"`
	MaxLifetime time.Duration `default:"300s"`
	IdleTimeout time.Duration `default:"60s"`
}

// defaultPoolConfig returns default configuration for connection pool
func defaultPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxIdle:     5,
		MaxLifetime: 5 * time.Minute,
		IdleTimeout: 1 * time.Minute,
	}
}

// NewConnectionPool creates a new SSH connection pool
func NewConnectionPool(cfg *ConnectionPoolConfig) *ConnectionPool {
	if cfg == nil {
		cfg = defaultPoolConfig()
	}

	pool := &ConnectionPool{
		pool:        make(map[string]*PooledConnection),
		maxIdle:     cfg.MaxIdle,
		maxLifetime: cfg.MaxLifetime,
		idleTimeout: cfg.IdleTimeout,
	}

	// Start cleanup goroutine
	go pool.cleanupLoop()

	return pool
}

// GetConnection retrieves a connection from the pool or creates a new one
func (p *ConnectionPool) GetConnection(ctx context.Context, host string, hostConfig config.Host, debug bool) (*PooledConnection, error) {
	hostKey := fmt.Sprintf("%s@%s", hostConfig.Username, hostConfig.Host)

	p.poolLock.RLock()
	conn, exists := p.pool[hostKey]
	p.poolLock.RUnlock()

	if exists && conn.isUsable() {
		conn.useLock.Lock()
		if !conn.inUse {
			conn.inUse = true
			conn.lastUsed = time.Now()
			conn.useLock.Unlock()
			
			logger.Debug("Reusing SSH connection from pool", map[string]interface{}{
				"host": hostKey,
				"age":  time.Since(conn.createdAt).String(),
			})
			
			return conn, nil
		}
		conn.useLock.Unlock()
	}

	// Create new connection
	client, err := NewClient(host, hostConfig, debug)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH client: %v", err)
	}

	if err := client.ConnectWithContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect SSH client: %v", err)
	}

	pooledConn := &PooledConnection{
		client:    client.client,
		host:      hostKey,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		inUse:     true,
	}

	// Add to pool
	p.poolLock.Lock()
	// Remove old connection if it exists
	if oldConn, exists := p.pool[hostKey]; exists {
		go oldConn.close()
	}
	p.pool[hostKey] = pooledConn
	p.poolLock.Unlock()

	logger.Debug("Created new SSH connection", map[string]interface{}{
		"host": hostKey,
	})

	return pooledConn, nil
}

// ReleaseConnection returns a connection to the pool
func (p *ConnectionPool) ReleaseConnection(conn *PooledConnection) {
	if conn == nil {
		return
	}

	conn.useLock.Lock()
	conn.inUse = false
	conn.lastUsed = time.Now()
	conn.useLock.Unlock()

	logger.Debug("Released SSH connection to pool", map[string]interface{}{
		"host": conn.host,
	})
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	for host, conn := range p.pool {
		conn.close()
		logger.Debug("Closed pooled SSH connection", map[string]interface{}{
			"host": host,
		})
	}
	p.pool = make(map[string]*PooledConnection)
}

// Stats returns statistics about the connection pool
func (p *ConnectionPool) Stats() map[string]interface{} {
	p.poolLock.RLock()
	defer p.poolLock.RUnlock()

	inUse := 0
	idle := 0
	total := len(p.pool)

	for _, conn := range p.pool {
		conn.useLock.Lock()
		if conn.inUse {
			inUse++
		} else {
			idle++
		}
		conn.useLock.Unlock()
	}

	return map[string]interface{}{
		"total_connections": total,
		"in_use":           inUse,
		"idle":             idle,
		"max_idle":         p.maxIdle,
		"max_lifetime":     p.maxLifetime.String(),
		"idle_timeout":     p.idleTimeout.String(),
	}
}

// cleanupLoop periodically cleans up expired connections
func (p *ConnectionPool) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		p.cleanup()
	}
}

// cleanup removes expired and idle connections
func (p *ConnectionPool) cleanup() {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	now := time.Now()
	toRemove := make([]string, 0)

	for host, conn := range p.pool {
		conn.useLock.Lock()
		shouldRemove := false

		// Check if connection is too old
		if now.Sub(conn.createdAt) > p.maxLifetime {
			shouldRemove = true
			logger.Debug("Removing SSH connection due to max lifetime", map[string]interface{}{
				"host": host,
				"age":  now.Sub(conn.createdAt).String(),
			})
		}

		// Check if idle connection has timed out
		if !conn.inUse && now.Sub(conn.lastUsed) > p.idleTimeout {
			shouldRemove = true
			logger.Debug("Removing idle SSH connection", map[string]interface{}{
				"host":      host,
				"idle_time": now.Sub(conn.lastUsed).String(),
			})
		}

		conn.useLock.Unlock()

		if shouldRemove {
			toRemove = append(toRemove, host)
		}
	}

	// Remove expired connections
	for _, host := range toRemove {
		if conn, exists := p.pool[host]; exists {
			go conn.close()
			delete(p.pool, host)
		}
	}
}

// isUsable checks if a pooled connection is still usable
func (pc *PooledConnection) isUsable() bool {
	if pc.client == nil {
		return false
	}

	// Check if connection is still alive by sending a simple request
	session, err := pc.client.NewSession()
	if err != nil {
		return false
	}
	session.Close()

	return true
}

// close closes the underlying SSH connection
func (pc *PooledConnection) close() {
	if pc.client != nil {
		pc.client.Close()
	}
}

// ExecWithContext executes a command using the pooled connection
func (pc *PooledConnection) ExecWithContext(ctx context.Context, task config.Task, debug bool) (int, string, error) {
	if pc.client == nil {
		return -1, "", fmt.Errorf("connection is not available")
	}

	session, err := pc.client.NewSession()
	if err != nil {
		return -1, "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var stdout, stderr strings.Builder
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Enable pseudo-terminal if AskPass is set
	if task.AskPass {
		session.RequestPty("xterm", 80, 24, ssh.TerminalModes{})
	}

	// Prepend directory change if specified
	cmd := task.Cmd
	if task.Dir != "" {
		cmd = fmt.Sprintf("cd %s && %s", task.Dir, task.Cmd)
	}

	if debug {
		logger.Debug("Executing SSH command", map[string]interface{}{
			"host":    pc.host,
			"command": cmd,
		})
	}

	// Create a channel to receive the result
	type result struct {
		err error
	}
	resultChan := make(chan result, 1)

	// Run command in a goroutine
	go func() {
		err := session.Run(cmd)
		resultChan <- result{err: err}
	}()

	// Wait for either command completion or context cancellation
	select {
	case res := <-resultChan:
		output := stdout.String() + stderr.String()

		if res.err != nil {
			// Gracefully handle remote command exit codes
			if exitErr, ok := res.err.(*ssh.ExitError); ok {
				return exitErr.ExitStatus(), output, nil
			}
			return -1, output, res.err
		}
		return 0, output, nil

	case <-ctx.Done():
		// Context was cancelled or timed out
		return -1, "", fmt.Errorf("command execution cancelled: %v", ctx.Err())
	}
}