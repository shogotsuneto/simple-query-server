package middleware

import (
	"fmt"
	"net/http"
)

// HTTPHeaderConfig represents the configuration for HTTP header middleware
type HTTPHeaderConfig struct {
	Header    string `yaml:"header"`    // Name of the HTTP header to extract
	Parameter string `yaml:"parameter"` // Name of the SQL parameter to set
	Required  bool   `yaml:"required"`  // Whether the header is required
}

// HTTPHeaderMiddleware extracts values from HTTP headers and makes them available as SQL parameters
type HTTPHeaderMiddleware struct {
	config HTTPHeaderConfig
}

// NewHTTPHeaderMiddleware creates a new HTTP header middleware
func NewHTTPHeaderMiddleware(config HTTPHeaderConfig) *HTTPHeaderMiddleware {
	return &HTTPHeaderMiddleware{
		config: config,
	}
}

// Wrap wraps an http.HandlerFunc with this middleware
func (m *HTTPHeaderMiddleware) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		headerValue := r.Header.Get(m.config.Header)

		if headerValue == "" {
			if m.config.Required {
				http.Error(w, fmt.Sprintf("required header '%s' is missing", m.config.Header), http.StatusBadRequest)
				return
			}
			// If not required and missing, just continue without adding the parameter
			next.ServeHTTP(w, r)
			return
		}

		// Get existing middleware parameters from context
		params := GetMiddlewareParams(r)
		
		// Add the header value as a parameter
		params[m.config.Parameter] = headerValue

		// Set updated parameters in context and continue
		r = SetMiddlewareParams(r, params)
		next.ServeHTTP(w, r)
	}
}

// Name returns the name of this middleware
func (m *HTTPHeaderMiddleware) Name() string {
	return fmt.Sprintf("http-header(%s->%s)", m.config.Header, m.config.Parameter)
}
