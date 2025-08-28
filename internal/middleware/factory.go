package middleware

import (
	"fmt"
	
	"gopkg.in/yaml.v3"
	"github.com/shogotsuneto/simple-query-server/internal/config"
)

// CreateMiddleware creates a middleware instance from configuration
func CreateMiddleware(middlewareConfig config.MiddlewareConfig) (Middleware, error) {
	switch middlewareConfig.Type {
	case "http-header":
		return createHTTPHeaderMiddleware(middlewareConfig.Config)
	default:
		return nil, fmt.Errorf("unknown middleware type: %s", middlewareConfig.Type)
	}
}

// createHTTPHeaderMiddleware creates an HTTP header middleware from config
func createHTTPHeaderMiddleware(configMap map[string]interface{}) (Middleware, error) {
	// Convert the config map to YAML and back to get proper type conversion
	yamlData, err := yaml.Marshal(configMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal http-header config: %w", err)
	}
	
	var httpHeaderConfig HTTPHeaderConfig
	if err := yaml.Unmarshal(yamlData, &httpHeaderConfig); err != nil {
		return nil, fmt.Errorf("failed to parse http-header config: %w", err)
	}
	
	// Validate required fields
	if httpHeaderConfig.Header == "" {
		return nil, fmt.Errorf("http-header middleware requires 'header' field")
	}
	if httpHeaderConfig.Parameter == "" {
		return nil, fmt.Errorf("http-header middleware requires 'parameter' field")
	}
	
	return NewHTTPHeaderMiddleware(httpHeaderConfig), nil
}

// CreateMiddlewareChain creates a middleware chain from server configuration
func CreateMiddlewareChain(serverConfig *config.ServerConfig) (Chain, error) {
	if serverConfig == nil || len(serverConfig.Middleware) == 0 {
		return Chain{}, nil
	}
	
	chain := make(Chain, 0, len(serverConfig.Middleware))
	
	for i, middlewareConfig := range serverConfig.Middleware {
		middleware, err := CreateMiddleware(middlewareConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create middleware at index %d: %w", i, err)
		}
		chain = append(chain, middleware)
	}
	
	return chain, nil
}