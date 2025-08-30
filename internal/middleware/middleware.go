package middleware

import (
	"context"
	"log"
	"net/http"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const (
	// MiddlewareParamsKey is the context key for middleware parameters
	MiddlewareParamsKey contextKey = "middleware_params"
)

// Middleware represents a middleware that can wrap HTTP handlers
type Middleware interface {
	// Wrap wraps an http.HandlerFunc with this middleware
	Wrap(next http.HandlerFunc) http.HandlerFunc

	// Name returns the name of the middleware for logging/debugging
	Name() string
}

// CloseableMiddleware represents a middleware that needs cleanup
type CloseableMiddleware interface {
	Middleware
	// Close cleans up resources used by the middleware
	Close() error
}

// HealthChecker represents a middleware that can report its health status
type HealthChecker interface {
	// IsHealthy returns true if the middleware is healthy and can process requests properly
	IsHealthy() bool
	// HealthCheckEnabled returns true if health checking is enabled for this middleware
	HealthCheckEnabled() bool
}

// Chain represents a chain of middleware to be executed
type Chain []Middleware

// Wrap wraps an http.HandlerFunc with all middleware in the chain
func (c Chain) Wrap(handler http.HandlerFunc) http.HandlerFunc {
	// Wrap middleware in reverse order so the first middleware in the slice
	// is the outermost middleware (executes first)
	for i := len(c) - 1; i >= 0; i-- {
		handler = c[i].Wrap(handler)
	}
	return handler
}

// Close closes all closeable middleware in the chain
func (c Chain) Close() error {
	for _, middleware := range c {
		if closeable, ok := middleware.(CloseableMiddleware); ok {
			if err := closeable.Close(); err != nil {
				// Log error but continue closing other middleware
				log.Printf("Error closing middleware %s: %v", middleware.Name(), err)
			}
		}
	}
	return nil
}

// GetMiddlewareParams extracts middleware parameters from the request context
func GetMiddlewareParams(r *http.Request) map[string]interface{} {
	if params, ok := r.Context().Value(MiddlewareParamsKey).(map[string]interface{}); ok {
		return params
	}
	return make(map[string]interface{})
}

// SetMiddlewareParams sets middleware parameters in the request context
func SetMiddlewareParams(r *http.Request, params map[string]interface{}) *http.Request {
	ctx := context.WithValue(r.Context(), MiddlewareParamsKey, params)
	return r.WithContext(ctx)
}
