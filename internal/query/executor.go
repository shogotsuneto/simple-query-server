package query

import (
	"fmt"
	"log"
	"strings"

	"github.com/shogotsuneto/simple-query-server/internal/config"
)

// Executor handles query execution against the database
type Executor struct {
	dbConfig *config.DatabaseConfig
}

// NewExecutor creates a new query executor
func NewExecutor(dbConfig *config.DatabaseConfig) *Executor {
	return &Executor{
		dbConfig: dbConfig,
	}
}

// Execute executes a query with the given parameters
func (e *Executor) Execute(queryConfig config.Query, params map[string]interface{}) ([]map[string]interface{}, error) {
	log.Printf("Executing query: %s", queryConfig.SQL)
	log.Printf("Parameters: %+v", params)

	// TODO: Implement actual database connection and query execution
	// This is a placeholder implementation that returns mock data
	
	// Validate parameters
	if err := e.validateParameters(queryConfig, params); err != nil {
		return nil, err
	}

	// Mock response based on query type
	return e.generateMockResponse(queryConfig, params)
}

// validateParameters validates that required parameters are provided with correct types
func (e *Executor) validateParameters(queryConfig config.Query, params map[string]interface{}) error {
	for _, param := range queryConfig.Params {
		value, exists := params[param.Name]
		if !exists {
			return fmt.Errorf("required parameter '%s' is missing", param.Name)
		}

		// Basic type validation
		switch param.Type {
		case "int":
			switch v := value.(type) {
			case int, int32, int64, float64:
				// JSON numbers are parsed as float64, so we accept them for int parameters
			default:
				return fmt.Errorf("parameter '%s' must be an integer, got %T", param.Name, v)
			}
		case "string":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("parameter '%s' must be a string, got %T", param.Name, value)
			}
		case "float":
			switch value.(type) {
			case float32, float64, int, int32, int64:
				// Accept numeric types for float parameters
			default:
				return fmt.Errorf("parameter '%s' must be a number, got %T", param.Name, value)
			}
		}
	}
	return nil
}

// generateMockResponse generates mock data for demonstration purposes
func (e *Executor) generateMockResponse(queryConfig config.Query, params map[string]interface{}) ([]map[string]interface{}, error) {
	// This is a placeholder that generates mock responses based on the SQL query pattern
	sql := strings.ToLower(queryConfig.SQL)
	
	// Handle specific queries from our example configuration
	if strings.Contains(sql, "users") && strings.Contains(sql, "where id =") {
		// Mock response for get_user_by_id type queries
		if id, exists := params["id"]; exists {
			return []map[string]interface{}{
				{
					"id":    id,
					"name":  fmt.Sprintf("User %v", id),
					"email": fmt.Sprintf("user%v@example.com", id),
				},
			}, nil
		}
	}
	
	if strings.Contains(sql, "users") && strings.Contains(sql, "like") {
		// Mock response for search_users type queries
		return []map[string]interface{}{
			{
				"id":   1,
				"name": "Alice Smith",
			},
			{
				"id":   2,
				"name": "Alice Johnson",
			},
		}, nil
	}
	
	if strings.Contains(sql, "users") && strings.Contains(sql, "active = true") {
		// Mock response for get_all_active_users
		return []map[string]interface{}{
			{
				"id":    1,
				"name":  "Active User 1",
				"email": "user1@example.com",
			},
			{
				"id":    2,
				"name":  "Active User 2", 
				"email": "user2@example.com",
			},
		}, nil
	}

	// Default mock response for any other query
	return []map[string]interface{}{
		{
			"message": "Query executed successfully (mock data)",
			"sql":     queryConfig.SQL,
			"params":  params,
		},
	}, nil
}

// TODO: Implement actual database connection methods
// - func (e *Executor) Connect() error
// - func (e *Executor) Close() error  
// - func (e *Executor) executeSQL(sql string, params map[string]interface{}) ([]map[string]interface{}, error)
// 
// Example database drivers to consider:
// - database/sql with driver for PostgreSQL: "github.com/lib/pq"
// - database/sql with driver for MySQL: "github.com/go-sql-driver/mysql"
// - database/sql with driver for SQLite: "github.com/mattn/go-sqlite3"