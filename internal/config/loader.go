package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DatabaseConfig represents the database configuration
type DatabaseConfig struct {
	Type        string            `yaml:"type"`        // e.g., "postgres", "mysql", "sqlite"
	DSN         string            `yaml:"dsn"`         // Data Source Name
	Credentials map[string]string `yaml:"credentials"` // Additional credentials if needed
}

// QueryParam represents a parameter for a query
type QueryParam struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"` // "int", "string", "float", etc.
}

// Query represents a single query configuration
type Query struct {
	SQL    string       `yaml:"sql"`
	Params []QueryParam `yaml:"params"`
}

// QueriesConfig represents the queries configuration
type QueriesConfig struct {
	Queries map[string]Query `yaml:"queries"`
}

// LoadDatabaseConfig loads database configuration from a YAML file
func LoadDatabaseConfig(path string) (*DatabaseConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read database config file %s: %w", path, err)
	}

	var config DatabaseConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse database config YAML: %w", err)
	}

	// Basic validation
	if config.Type == "" {
		return nil, fmt.Errorf("database type is required")
	}
	if config.DSN == "" {
		return nil, fmt.Errorf("database DSN is required")
	}

	return &config, nil
}

// LoadQueriesConfig loads queries configuration from a YAML file
func LoadQueriesConfig(path string) (*QueriesConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read queries config file %s: %w", path, err)
	}

	var config QueriesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse queries config YAML: %w", err)
	}

	// Basic validation
	if len(config.Queries) == 0 {
		return nil, fmt.Errorf("at least one query must be defined")
	}

	for name, query := range config.Queries {
		if query.SQL == "" {
			return nil, fmt.Errorf("query %s must have SQL defined", name)
		}
	}

	return &config, nil
}