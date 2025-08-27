package query

import (
	"fmt"

	"github.com/shogotsuneto/simple-query-server/internal/config"
)

// QueryExecutor defines the interface for database query execution.
// This interface enables pluggable database support by abstracting
// database-specific implementation details behind a common contract.
//
// Implementations should handle:
// - Database connection management
// - Parameter validation and binding
// - SQL execution and result processing
// - Database-specific syntax and data type handling
// - Proper error handling and resource cleanup
//
// Current implementations:
// - PostgreSQLExecutor: Full PostgreSQL support
type QueryExecutor interface {
	// Execute runs a query with the given parameters and returns results as rows of key-value pairs.
	// Parameters are validated according to the query configuration before execution.
	Execute(queryConfig config.Query, params map[string]interface{}) ([]map[string]interface{}, error)

	// Close releases database resources and closes the connection.
	// Should be called when the executor is no longer needed.
	Close() error
}

// NewQueryExecutor creates a new query executor based on database type
func NewQueryExecutor(dbConfig *config.DatabaseConfig) (QueryExecutor, error) {
	// Database configuration is required
	if dbConfig.DSN == "" || dbConfig.Type == "" {
		return nil, fmt.Errorf("database configuration is required: both type and DSN must be provided")
	}

	switch dbConfig.Type {
	case "postgres":
		return NewPostgreSQLExecutor(dbConfig)
	// case "mysql":
	// 	return NewMySQLExecutor(dbConfig)
	default:
		return nil, fmt.Errorf("database type '%s' is not supported yet (supported types: postgres)", dbConfig.Type)
	}
}
