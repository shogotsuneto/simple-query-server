package query

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/shogotsuneto/simple-query-server/internal/config"
)

// PostgreSQLExecutor handles query execution against PostgreSQL databases
type PostgreSQLExecutor struct {
	dbConfig *config.DatabaseConfig
	db       *sql.DB
}

// NewPostgreSQLExecutor creates a new PostgreSQL query executor
func NewPostgreSQLExecutor(dbConfig *config.DatabaseConfig) (*PostgreSQLExecutor, error) {
	executor := &PostgreSQLExecutor{
		dbConfig: dbConfig,
	}

	// Connect to database - fail if connection fails
	if err := executor.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL database: %w", err)
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

	// Execute SQL query against the database
	if e.db == nil {
		return nil, fmt.Errorf("no database connection available")
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
