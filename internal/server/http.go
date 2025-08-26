package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/shogotsuneto/simple-query-server/internal/config"
	"github.com/shogotsuneto/simple-query-server/internal/query"
)

// Server represents the HTTP server
type Server struct {
	dbConfig      *config.DatabaseConfig
	queriesConfig *config.QueriesConfig
	executor      *query.Executor
}

// Response represents the JSON response structure
type Response struct {
	Rows  []map[string]interface{} `json:"rows,omitempty"`
	Error string                   `json:"error,omitempty"`
}

// New creates a new Server instance
func New(dbConfig *config.DatabaseConfig, queriesConfig *config.QueriesConfig) *Server {
	executor := query.NewExecutor(dbConfig)
	return &Server{
		dbConfig:      dbConfig,
		queriesConfig: queriesConfig,
		executor:      executor,
	}
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

	response := map[string]string{
		"status": "healthy",
	}

	w.Header().Set("Content-Type", "application/json")
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

	// Execute the query
	rows, err := s.executor.Execute(queryConfig, params)
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