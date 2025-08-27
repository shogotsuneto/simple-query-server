package query

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/shogotsuneto/simple-query-server/internal/config"
)

// PostgreSQLExecutor handles query execution against PostgreSQL databases
type PostgreSQLExecutor struct {
	dbConfig *config.DatabaseConfig
	db       *sql.DB
}

const (
	// Connection retry configuration
	maxRetries = 5
	baseDelay  = 1 * time.Second
	maxDelay   = 30 * time.Second
)

// NewPostgreSQLExecutor creates a new PostgreSQL query executor
func NewPostgreSQLExecutor(dbConfig *config.DatabaseConfig) (*PostgreSQLExecutor, error) {
	executor := &PostgreSQLExecutor{
		dbConfig: dbConfig,
	}

	// Try to connect initially, but don't fail if it doesn't work
	// The connection will be retried later when needed
	err := executor.connect()
	if err != nil {
		log.Printf("Initial database connection failed: %v", err)
		log.Printf("Server will continue starting, database connection will be retried when needed")
	}

	return executor, nil
}

// Execute executes a query with the given parameters
func (e *PostgreSQLExecutor) Execute(queryConfig config.Query, params map[string]interface{}) ([]map[string]interface{}, error) {
	log.Printf("Executing PostgreSQL query: %s", queryConfig.SQL)
	log.Printf("Parameters: %+v", params)

	// Validate parameters
	if err := e.validateParameters(queryConfig, params); err != nil {
		return nil, err
	}

	// Ensure we have a database connection
	if err := e.ensureConnection(); err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	return e.executeSQL(queryConfig.SQL, params)
}

// validateParameters validates that required parameters are provided with correct types
func (e *PostgreSQLExecutor) validateParameters(queryConfig config.Query, params map[string]interface{}) error {
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

// connect establishes a connection to the PostgreSQL database
func (e *PostgreSQLExecutor) connect() error {
	var err error
	e.db, err = sql.Open("postgres", e.dbConfig.DSN)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Test the connection
	if err := e.db.Ping(); err != nil {
		e.db.Close()
		e.db = nil
		return fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	log.Printf("Successfully connected to PostgreSQL database")
	return nil
}

// ensureConnection ensures we have a working database connection, with retry logic
func (e *PostgreSQLExecutor) ensureConnection() error {
	// If we have a connection, test it first
	if e.db != nil {
		if err := e.db.Ping(); err == nil {
			return nil // Connection is good
		} else {
			// Connection is bad, clean it up
			log.Printf("Existing database connection failed ping: %v", err)
			e.db.Close()
			e.db = nil
		}
	}

	// Try to connect with retries
	delay := baseDelay
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Attempting database connection (attempt %d/%d)...", attempt, maxRetries)

		if err := e.connect(); err != nil {
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

// IsHealthy returns true if the database connection is healthy
func (e *PostgreSQLExecutor) IsHealthy() bool {
	if e.db == nil {
		// Try to connect without retries for health check
		if err := e.connect(); err != nil {
			return false
		}
	}

	// Quick ping to verify connection is still alive
	if err := e.db.Ping(); err != nil {
		log.Printf("Database health check failed: %v", err)
		e.db.Close()
		e.db = nil
		return false
	}

	return true
}

// Close closes the database connection
func (e *PostgreSQLExecutor) Close() error {
	if e.db != nil {
		err := e.db.Close()
		e.db = nil
		return err
	}
	return nil
}

// executeSQL executes a SQL query against the PostgreSQL database
func (e *PostgreSQLExecutor) executeSQL(sql string, params map[string]interface{}) ([]map[string]interface{}, error) {
	// Convert :param syntax to PostgreSQL $1, $2, ... syntax
	convertedSQL, args, err := e.convertSQLParameters(sql, params)
	if err != nil {
		return nil, fmt.Errorf("failed to convert SQL parameters: %w", err)
	}

	log.Printf("Executing PostgreSQL SQL: %s", convertedSQL)
	log.Printf("Arguments: %+v", args)

	rows, err := e.db.Query(convertedSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute PostgreSQL query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get column names: %w", err)
	}

	var results []map[string]interface{}

	for rows.Next() {
		// Create slice to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert to map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for better JSON serialization
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return results, nil
}

// convertSQLParameters converts :param syntax to PostgreSQL $1, $2, ... syntax
func (e *PostgreSQLExecutor) convertSQLParameters(sql string, params map[string]interface{}) (string, []interface{}, error) {
	// Find all :param references in the SQL
	re := regexp.MustCompile(`:(\w+)`)
	matches := re.FindAllStringSubmatch(sql, -1)

	if len(matches) == 0 {
		// No parameters to convert
		return sql, []interface{}{}, nil
	}

	paramOrder := make([]string, 0, len(matches))
	paramSet := make(map[string]bool)

	// Extract unique parameter names in order of appearance
	for _, match := range matches {
		paramName := match[1]
		if !paramSet[paramName] {
			paramOrder = append(paramOrder, paramName)
			paramSet[paramName] = true
		}
	}

	// Build args slice and replace parameters
	args := make([]interface{}, len(paramOrder))
	convertedSQL := sql

	for i, paramName := range paramOrder {
		value, exists := params[paramName]
		if !exists {
			return "", nil, fmt.Errorf("parameter '%s' referenced in SQL but not provided", paramName)
		}
		args[i] = value

		// Replace all occurrences of this parameter
		paramPlaceholder := fmt.Sprintf(":%s", paramName)
		pgPlaceholder := fmt.Sprintf("$%d", i+1)
		convertedSQL = strings.ReplaceAll(convertedSQL, paramPlaceholder, pgPlaceholder)
	}

	return convertedSQL, args, nil
}
