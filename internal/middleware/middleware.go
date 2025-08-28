package middleware

import (
	"context"
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
