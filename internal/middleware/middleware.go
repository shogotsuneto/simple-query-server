package middleware

import (
	"net/http"
)

// Middleware represents a middleware that can process HTTP requests
type Middleware interface {
	// Process processes the request and may modify the parameters
	// Returns the modified parameters and any error
	Process(r *http.Request, params map[string]interface{}) (map[string]interface{}, error)

	// Name returns the name of the middleware for logging/debugging
	Name() string
}

// Chain represents a chain of middleware to be executed
type Chain []Middleware

// Process executes all middleware in the chain in order
func (c Chain) Process(r *http.Request, params map[string]interface{}) (map[string]interface{}, error) {
	result := params
	if result == nil {
		result = make(map[string]interface{})
	}

	var err error
	for _, middleware := range c {
		result, err = middleware.Process(r, result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
