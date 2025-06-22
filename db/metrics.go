package db

import (
	"database/sql"
	"log"
	"sync/atomic"
	"time"
)

// Metrics tracks database operation statistics
type Metrics struct {
	QueryCount    int64
	ErrorCount    int64
	TotalDuration int64 // in nanoseconds
	OpenConns     int32
	IdleConns     int32
}

// MetricsDB wraps a sql.DB with performance monitoring
type MetricsDB struct {
	*sql.DB
	metrics *Metrics
}

// NewMetricsDB creates a new database wrapper with metrics tracking
func NewMetricsDB(db *sql.DB) *MetricsDB {
	return &MetricsDB{
		DB:      db,
		metrics: &Metrics{},
	}
}

// GetMetrics returns a copy of current metrics
func (m *MetricsDB) GetMetrics() Metrics {
	return Metrics{
		QueryCount:    atomic.LoadInt64(&m.metrics.QueryCount),
		ErrorCount:    atomic.LoadInt64(&m.metrics.ErrorCount),
		TotalDuration: atomic.LoadInt64(&m.metrics.TotalDuration),
		OpenConns:     atomic.LoadInt32(&m.metrics.OpenConns),
		IdleConns:     atomic.LoadInt32(&m.metrics.IdleConns),
	}
}

// UpdateConnectionStats updates connection pool statistics
func (m *MetricsDB) UpdateConnectionStats() {
	stats := m.DB.Stats()
	atomic.StoreInt32(&m.metrics.OpenConns, int32(stats.OpenConnections))
	atomic.StoreInt32(&m.metrics.IdleConns, int32(stats.Idle))
}

// Query wraps sql.DB.Query with metrics
func (m *MetricsDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := m.DB.Query(query, args...)
	duration := time.Since(start)
	
	atomic.AddInt64(&m.metrics.QueryCount, 1)
	atomic.AddInt64(&m.metrics.TotalDuration, duration.Nanoseconds())
	
	if err != nil {
		atomic.AddInt64(&m.metrics.ErrorCount, 1)
		log.Printf("DB Query Error: %v | Query: %s", err, query)
	}
	
	m.UpdateConnectionStats()
	return rows, err
}

// QueryRow wraps sql.DB.QueryRow with metrics
func (m *MetricsDB) QueryRow(query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := m.DB.QueryRow(query, args...)
	duration := time.Since(start)
	
	atomic.AddInt64(&m.metrics.QueryCount, 1)
	atomic.AddInt64(&m.metrics.TotalDuration, duration.Nanoseconds())
	
	m.UpdateConnectionStats()
	return row
}

// Exec wraps sql.DB.Exec with metrics
func (m *MetricsDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := m.DB.Exec(query, args...)
	duration := time.Since(start)
	
	atomic.AddInt64(&m.metrics.QueryCount, 1)
	atomic.AddInt64(&m.metrics.TotalDuration, duration.Nanoseconds())
	
	if err != nil {
		atomic.AddInt64(&m.metrics.ErrorCount, 1)
		log.Printf("DB Exec Error: %v | Query: %s", err, query)
	}
	
	m.UpdateConnectionStats()
	return result, err
}

// Begin wraps sql.DB.Begin with metrics
func (m *MetricsDB) Begin() (*sql.Tx, error) {
	start := time.Now()
	tx, err := m.DB.Begin()
	duration := time.Since(start)
	
	atomic.AddInt64(&m.metrics.QueryCount, 1)
	atomic.AddInt64(&m.metrics.TotalDuration, duration.Nanoseconds())
	
	if err != nil {
		atomic.AddInt64(&m.metrics.ErrorCount, 1)
		log.Printf("DB Begin Error: %v", err)
	}
	
	m.UpdateConnectionStats()
	return tx, err
}

// LogMetrics logs current database metrics
func (m *MetricsDB) LogMetrics() {
	metrics := m.GetMetrics()
	avgDuration := float64(0)
	if metrics.QueryCount > 0 {
		avgDuration = float64(metrics.TotalDuration) / float64(metrics.QueryCount) / 1e6 // Convert to milliseconds
	}
	
	log.Printf("DB Metrics - Queries: %d, Errors: %d, Avg Duration: %.2fms, Open Conns: %d, Idle Conns: %d",
		metrics.QueryCount,
		metrics.ErrorCount,
		avgDuration,
		metrics.OpenConns,
		metrics.IdleConns,
	)
}