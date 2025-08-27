package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/shogotsuneto/simple-query-server/internal/config"
	"github.com/shogotsuneto/simple-query-server/internal/query"
)

const (
	// Connection retry configuration
	maxRetries = 5
	baseDelay  = 1 * time.Second
	maxDelay   = 30 * time.Second
	// Health check interval
	healthCheckInterval = 30 * time.Second
)

// Server represents the HTTP server with database connection management
type Server struct {
	dbConfig      *config.DatabaseConfig
	queriesConfig *config.QueriesConfig
	executor      query.QueryExecutor
	db            *sql.DB
	healthy       int64                // atomic boolean for health status
	cancel        context.CancelFunc   // for stopping the health check goroutine
}

// Response represents the JSON response structure
type Response struct {
	Rows  []map[string]interface{} `json:"rows,omitempty"`
	Error string                   `json:"error,omitempty"`
}

// New creates a new Server instance with database connection management
func New(dbConfig *config.DatabaseConfig, queriesConfig *config.QueriesConfig) (*Server, error) {
	executor, err := query.NewQueryExecutor(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create query executor: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	server := &Server{
		dbConfig:      dbConfig,
		queriesConfig: queriesConfig,
		executor:      executor,
		cancel:        cancel,
	}

	// Try to connect initially, but don't fail if it doesn't work
	// The connection will be retried later when needed
	err = server.connect()
	if err != nil {
		log.Printf("Initial database connection failed: %v", err)
		log.Printf("Server will continue starting, database connection will be retried when needed")
		atomic.StoreInt64(&server.healthy, 0) // unhealthy
	} else {
		atomic.StoreInt64(&server.healthy, 1) // healthy
	}

	// Start background health monitoring
	go server.healthMonitor(ctx)

	return server, nil
}

// Start starts the HTTP server on the specified port
func (s *Server) Start(port string) error {
	http.HandleFunc("/", s.handleRoot)
	http.HandleFunc("/health", s.handleHealth)
	http.HandleFunc("/queries", s.handleListQueries)
	http.HandleFunc("/query/", s.handleQuery)

	addr := ":" + port
	log.Printf("Server starting on %s", addr)
	log.Printf("Available endpoints:")
	log.Printf("  GET  /health       - Health check")
	log.Printf("  GET  /queries      - List available queries")
	log.Printf("  POST /query/{name} - Execute a query")

	return http.ListenAndServe(addr, nil)
}

// connect establishes a connection to the database
func (s *Server) connect() error {
	var err error
	s.db, err = sql.Open("postgres", s.dbConfig.DSN)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := s.db.Ping(); err != nil {
		s.db.Close()
		s.db = nil
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Successfully connected to database")
	return nil
}

// ensureConnection ensures we have a database connection with retry logic
func (s *Server) ensureConnection() error {
	// If we have a connection, assume it's good (health monitor handles health checks)
	if s.db != nil {
		return nil
	}

	// Try to connect with retries
	delay := baseDelay
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Attempting database connection (attempt %d/%d)...", attempt, maxRetries)

		if err := s.connect(); err != nil {
			log.Printf("Database connection attempt %d failed: %v", attempt, err)

			if attempt < maxRetries {
				log.Printf("Retrying in %v...", delay)
				time.Sleep(delay)
				// Exponential backoff with max delay
				delay = delay * 2
				if delay > maxDelay {
					delay = maxDelay
				}
			} else {
				return fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
			}
		} else {
			log.Printf("Database connection established successfully on attempt %d", attempt)
			return nil
		}
	}

	return fmt.Errorf("failed to connect after %d attempts", maxRetries)
}

// healthMonitor runs periodic health checks in the background
func (s *Server) healthMonitor(ctx context.Context) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check and updates the cached status
func (s *Server) performHealthCheck() {
	if s.db == nil {
		// Try to connect without retries for health check
		if err := s.connect(); err != nil {
			atomic.StoreInt64(&s.healthy, 0) // unhealthy
			return
		}
	}

	// Quick ping to verify connection is still alive
	if err := s.db.Ping(); err != nil {
		log.Printf("Database health check failed: %v", err)
		s.db.Close()
		s.db = nil
		atomic.StoreInt64(&s.healthy, 0) // unhealthy
	} else {
		atomic.StoreInt64(&s.healthy, 1) // healthy
	}
}

// IsHealthy returns the cached health status without performing a ping
func (s *Server) IsHealthy() bool {
	return atomic.LoadInt64(&s.healthy) == 1
}

// Close closes the database connection and stops the health monitor
func (s *Server) Close() error {
	// Stop the health monitor
	if s.cancel != nil {
		s.cancel()
	}
	
	// Close executor
	if err := s.executor.Close(); err != nil {
		log.Printf("Error closing executor: %v", err)
	}
	
	// Close database connection
	if s.db != nil {
		err := s.db.Close()
		s.db = nil
		atomic.StoreInt64(&s.healthy, 0) // unhealthy
		return err
	}
	return nil
}

// handleRoot handles requests to the root path
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	response := map[string]interface{}{
		"service": "simple-query-server",
		"status":  "running",
		"endpoints": map[string]string{
			"/health":       "GET - Health check",
			"/queries":      "GET - List available queries",
			"/query/{name}": "POST - Execute a query",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check database health using server's cached status
	dbHealthy := s.IsHealthy()

	var status string
	var statusCode int

	if dbHealthy {
		status = "healthy"
		statusCode = http.StatusOK
	} else {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	response := map[string]interface{}{
		"status": status,
		"database": map[string]bool{
			"connected": dbHealthy,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleListQueries handles requests to list available queries
func (s *Server) handleListQueries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	queries := make(map[string]interface{})
	for name, query := range s.queriesConfig.Queries {
		queries[name] = map[string]interface{}{
			"sql":    query.SQL,
			"params": query.Params,
		}
	}

	response := map[string]interface{}{
		"queries": queries,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleQuery handles query execution requests
func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract query name from path
	path := strings.TrimPrefix(r.URL.Path, "/query/")
	if path == "" {
		s.writeErrorResponse(w, "Query name is required", http.StatusBadRequest)
		return
	}

	// Find the query configuration
	queryConfig, exists := s.queriesConfig.Queries[path]
	if !exists {
		s.writeErrorResponse(w, fmt.Sprintf("Query '%s' not found", path), http.StatusNotFound)
		return
	}

	// Parse request body as JSON
	var params map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		s.writeErrorResponse(w, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}

	// Ensure we have a database connection
	if err := s.ensureConnection(); err != nil {
		log.Printf("Database connection error: %v", err)
		s.writeErrorResponse(w, fmt.Sprintf("Database connection failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Execute the query using the established connection
	rows, err := s.executor.Execute(s.db, queryConfig, params)
	if err != nil {
		log.Printf("Query execution error: %v", err)
		s.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send successful response
	response := Response{Rows: rows}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse writes an error response
func (s *Server) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := Response{Error: message}
	json.NewEncoder(w).Encode(response)
}
