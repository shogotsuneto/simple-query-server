package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/shogotsuneto/simple-query-server/internal/config"
)

const (
	// Connection retry configuration
	maxRetries = 5
	baseDelay  = 1 * time.Second
	maxDelay   = 30 * time.Second
	// Health check interval
	healthCheckInterval = 30 * time.Second
)

// PostgreSQLManager manages PostgreSQL database connections and health monitoring
type PostgreSQLManager struct {
	dbConfig *config.DatabaseConfig
	db       *sql.DB
	healthy  int64              // atomic boolean for health status
	cancel   context.CancelFunc // for stopping the health check goroutine
}

// NewPostgreSQLManager creates a new PostgreSQL database connection manager
func NewPostgreSQLManager(dbConfig *config.DatabaseConfig) (*PostgreSQLManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &PostgreSQLManager{
		dbConfig: dbConfig,
		cancel:   cancel,
	}

	// Start background connection management and health monitoring
	go manager.connectionManager(ctx)

	return manager, nil
}

// GetConnection returns the current database connection
func (m *PostgreSQLManager) GetConnection() *sql.DB {
	return m.db
}

// IsHealthy returns the cached health status
func (m *PostgreSQLManager) IsHealthy() bool {
	return atomic.LoadInt64(&m.healthy) == 1
}

// Close closes the database connection and stops the connection manager
func (m *PostgreSQLManager) Close() error {
	// Stop the connection manager
	if m.cancel != nil {
		m.cancel()
	}

	// Close database connection
	if m.db != nil {
		err := m.db.Close()
		m.db = nil
		atomic.StoreInt64(&m.healthy, 0) // unhealthy
		return err
	}
	return nil
}

// connectionManager handles connection establishment and health monitoring in background
func (m *PostgreSQLManager) connectionManager(ctx context.Context) {
	// Try initial connection
	m.tryConnect()

	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.performHealthCheck()
		}
	}
}

// tryConnect attempts to establish database connection with retries
func (m *PostgreSQLManager) tryConnect() {
	delay := baseDelay
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Attempting database connection (attempt %d/%d)...", attempt, maxRetries)

		if err := m.connect(); err != nil {
			log.Printf("Database connection attempt %d failed: %v", attempt, err)
			atomic.StoreInt64(&m.healthy, 0) // unhealthy

			if attempt < maxRetries {
				log.Printf("Retrying in %v...", delay)
				time.Sleep(delay)
				// Exponential backoff with max delay
				delay = delay * 2
				if delay > maxDelay {
					delay = maxDelay
				}
			} else {
				log.Printf("Failed to connect after %d attempts, will retry during health checks", maxRetries)
			}
		} else {
			log.Printf("Database connection established successfully on attempt %d", attempt)
			atomic.StoreInt64(&m.healthy, 1) // healthy
			return
		}
	}
}

// connect establishes a connection to the database
func (m *PostgreSQLManager) connect() error {
	var err error
	db, err := sql.Open("postgres", m.dbConfig.DSN)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	// Close old connection if exists
	if m.db != nil {
		m.db.Close()
	}

	m.db = db
	log.Printf("Successfully connected to PostgreSQL database")
	return nil
}

// performHealthCheck performs a health check and updates the cached status
func (m *PostgreSQLManager) performHealthCheck() {
	if m.db == nil {
		// No connection, try to establish one
		m.tryConnect()
		return
	}

	// Quick ping to verify connection is still alive
	if err := m.db.Ping(); err != nil {
		log.Printf("Database health check failed: %v", err)
		m.db.Close()
		m.db = nil
		atomic.StoreInt64(&m.healthy, 0) // unhealthy
		// Try to reconnect
		m.tryConnect()
	} else {
		atomic.StoreInt64(&m.healthy, 1) // healthy
	}
}
