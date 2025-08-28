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

// Process extracts the configured header value and adds it to parameters
func (m *HTTPHeaderMiddleware) Process(r *http.Request, params map[string]interface{}) (map[string]interface{}, error) {
	headerValue := r.Header.Get(m.config.Header)
	
	if headerValue == "" {
		if m.config.Required {
			return nil, fmt.Errorf("required header '%s' is missing", m.config.Header)
		}
		// If not required and missing, just return params unchanged
		return params, nil
	}
	
	// Make a copy of params to avoid modifying the original map
	result := make(map[string]interface{})
	for k, v := range params {
		result[k] = v
	}
	
	// Add the header value as a parameter
	result[m.config.Parameter] = headerValue
	
	return result, nil
}

// Name returns the name of this middleware
func (m *HTTPHeaderMiddleware) Name() string {
	return fmt.Sprintf("http-header(%s->%s)", m.config.Header, m.config.Parameter)
}