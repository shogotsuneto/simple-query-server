package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerJWKSMiddleware_RequiredFalse(t *testing.T) {
	// Test when required is false and no Authorization header is provided
	// NOTE: This test does not call the JWKS client because no Authorization header is present
	config := BearerJWKSConfig{
		JWKSURL:       "http://localhost:3000/.well-known/jwks.json",
		Required:      false,
		ClaimsMapping: map[string]string{"sub": "user_id"},
	}
	middleware := NewBearerJWKSMiddleware(config)

	var capturedParams map[string]interface{}
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		capturedParams = GetMiddlewareParams(r)
		w.WriteHeader(http.StatusOK)
	}

	// Wrap handler with middleware
	wrappedHandler := middleware.Wrap(testHandler)

	// Create request without Authorization header
	req, _ := http.NewRequest("POST", "/", nil)
	rr := httptest.NewRecorder()

	wrappedHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", rr.Code)
	}

	// Should continue without authentication when not required
	if len(capturedParams) != 0 {
		t.Errorf("Expected no parameters when no auth header provided, got %v", capturedParams)
	}
}

func TestBearerJWKSMiddleware_RequiredTrue(t *testing.T) {
	// Test when required is true and no Authorization header is provided
	// NOTE: This test does not call the JWKS client because no Authorization header is present
	config := BearerJWKSConfig{
		JWKSURL:       "http://localhost:3000/.well-known/jwks.json",
		Required:      true,
		ClaimsMapping: map[string]string{"sub": "user_id"},
	}
	middleware := NewBearerJWKSMiddleware(config)

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Wrap handler with middleware
	wrappedHandler := middleware.Wrap(testHandler)

	// Create request without Authorization header
	req, _ := http.NewRequest("POST", "/", nil)
	rr := httptest.NewRecorder()

	wrappedHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status Unauthorized, got %v", rr.Code)
	}
}

func TestBearerJWKSMiddleware_InvalidBearerFormat(t *testing.T) {
	// Test with invalid Bearer token format
	// NOTE: This test does not call the JWKS client because the Authorization header is not in Bearer format
	config := BearerJWKSConfig{
		JWKSURL:       "http://localhost:3000/.well-known/jwks.json",
		Required:      true,
		ClaimsMapping: map[string]string{"sub": "user_id"},
	}
	middleware := NewBearerJWKSMiddleware(config)

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Wrap handler with middleware
	wrappedHandler := middleware.Wrap(testHandler)

	// Create request with invalid Authorization header
	req, _ := http.NewRequest("POST", "/", nil)
	req.Header.Set("Authorization", "Basic invalid")
	rr := httptest.NewRecorder()

	wrappedHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status Unauthorized, got %v", rr.Code)
	}
}

func TestBearerJWKSMiddleware_EmptyBearerToken(t *testing.T) {
	// Test with empty Bearer token
	// NOTE: This test does not call the JWKS client because the Bearer token is empty
	config := BearerJWKSConfig{
		JWKSURL:       "http://localhost:3000/.well-known/jwks.json",
		Required:      true,
		ClaimsMapping: map[string]string{"sub": "user_id"},
	}
	middleware := NewBearerJWKSMiddleware(config)

	testHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Wrap handler with middleware
	wrappedHandler := middleware.Wrap(testHandler)

	// Create request with empty Bearer token
	req, _ := http.NewRequest("POST", "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rr := httptest.NewRecorder()

	wrappedHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status Unauthorized, got %v", rr.Code)
	}
}

func TestBearerJWKSMiddleware_Name(t *testing.T) {
	config := BearerJWKSConfig{
		JWKSURL:       "http://localhost:3000/.well-known/jwks.json",
		Required:      false,
		ClaimsMapping: map[string]string{"sub": "user_id"},
	}
	middleware := NewBearerJWKSMiddleware(config)

	expectedName := "bearer-jwks(http://localhost:3000/.well-known/jwks.json)"
	if middleware.Name() != expectedName {
		t.Errorf("Expected name %s, got %s", expectedName, middleware.Name())
	}
}

func TestCreateBearerJWKSMiddleware(t *testing.T) {
	// Test factory function
	configMap := map[string]interface{}{
		"jwks_url": "http://localhost:3000/.well-known/jwks.json",
		"required": true,
		"claims_mapping": map[string]interface{}{
			"sub":  "user_id",
			"role": "user_role",
		},
		"issuer":   "http://localhost:3000",
		"audience": "dev-api",
	}

	middleware, err := createBearerJWKSMiddleware(configMap)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	if middleware == nil {
		t.Fatal("Middleware is nil")
	}

	// Check that it implements the Middleware interface
	if _, ok := middleware.(*BearerJWKSMiddleware); !ok {
		t.Error("Middleware does not implement BearerJWKSMiddleware type")
	}
}

func TestCreateBearerJWKSMiddleware_MissingFields(t *testing.T) {
	// Test factory function with missing required fields
	testCases := []struct {
		name      string
		configMap map[string]interface{}
	}{
		{
			name: "missing jwks_url",
			configMap: map[string]interface{}{
				"required": true,
				"claims_mapping": map[string]interface{}{
					"sub": "user_id",
				},
			},
		},
		{
			name: "missing claims_mapping",
			configMap: map[string]interface{}{
				"jwks_url": "http://localhost:3000/.well-known/jwks.json",
				"required": true,
			},
		},
		{
			name: "empty claims_mapping",
			configMap: map[string]interface{}{
				"jwks_url":       "http://localhost:3000/.well-known/jwks.json",
				"required":       true,
				"claims_mapping": map[string]interface{}{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := createBearerJWKSMiddleware(tc.configMap)
			if err == nil {
				t.Error("Expected error for missing required field, but got none")
			}
		})
	}
}

func TestBearerJWKSMiddleware_OptionalNotRequiredWithInvalidToken(t *testing.T) {
	// Test when required is false and an invalid token is provided
	// NOTE: This test does not call the JWKS client because the token format is invalid (fails JWT parsing before JWKS lookup)
	config := BearerJWKSConfig{
		JWKSURL:       "http://localhost:3000/.well-known/jwks.json",
		Required:      false,
		ClaimsMapping: map[string]string{"sub": "user_id"},
	}
	middleware := NewBearerJWKSMiddleware(config)

	var capturedParams map[string]interface{}
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		capturedParams = GetMiddlewareParams(r)
		w.WriteHeader(http.StatusOK)
	}

	// Wrap handler with middleware
	wrappedHandler := middleware.Wrap(testHandler)

	// Create request with invalid token (will fail validation but should continue)
	req, _ := http.NewRequest("POST", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	rr := httptest.NewRecorder()

	wrappedHandler(rr, req)

	// Should continue without authentication when not required, even with invalid token
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK for optional auth with invalid token, got %v", rr.Code)
	}

	// Should have no parameters since token was invalid
	if len(capturedParams) != 0 {
		t.Errorf("Expected no parameters when invalid token provided to optional auth, got %v", capturedParams)
	}
}

// TestBearerJWKSMiddleware_HealthCheck tests the health check functionality
func TestBearerJWKSMiddleware_HealthCheck(t *testing.T) {
	t.Run("HealthCheckEnabledByDefault", func(t *testing.T) {
		config := BearerJWKSConfig{
			JWKSURL:       "http://localhost:3000/.well-known/jwks.json",
			Required:      false,
			ClaimsMapping: map[string]string{"sub": "user_id"},
			// EnableHealthCheck not set, should default to true
		}
		middleware := NewBearerJWKSMiddleware(config)

		if !middleware.HealthCheckEnabled() {
			t.Error("Expected health check to be enabled by default")
		}

		// Health check should work (though will be false due to no server)
		// We're just testing that the method exists and can be called
		_ = middleware.IsHealthy()
	})

	t.Run("HealthCheckExplicitlyEnabled", func(t *testing.T) {
		enabled := true
		config := BearerJWKSConfig{
			JWKSURL:           "http://localhost:3000/.well-known/jwks.json",
			Required:          false,
			ClaimsMapping:     map[string]string{"sub": "user_id"},
			EnableHealthCheck: &enabled,
		}
		middleware := NewBearerJWKSMiddleware(config)

		if !middleware.HealthCheckEnabled() {
			t.Error("Expected health check to be enabled when explicitly set to true")
		}
	})

	t.Run("HealthCheckExplicitlyDisabled", func(t *testing.T) {
		disabled := false
		config := BearerJWKSConfig{
			JWKSURL:           "http://localhost:3000/.well-known/jwks.json",
			Required:          false,
			ClaimsMapping:     map[string]string{"sub": "user_id"},
			EnableHealthCheck: &disabled,
		}
		middleware := NewBearerJWKSMiddleware(config)

		if middleware.HealthCheckEnabled() {
			t.Error("Expected health check to be disabled when explicitly set to false")
		}
	})

	t.Run("HealthCheckWithValidJWKS", func(t *testing.T) {
		// Mock JWKS server
		mockJWKS := `{
			"keys": [
				{
					"kty": "RSA",
					"kid": "test-key-1",
					"use": "sig",
					"n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISzIWzYr_W6UU9dwuW6TU0DjW0nQcaOLGOjQhGnOGKZ9CW7PDNE2J",
					"e": "AQAB"
				}
			]
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "max-age=3600") // 1 hour TTL
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockJWKS))
		}))
		defer server.Close()

		config := BearerJWKSConfig{
			JWKSURL:       server.URL,
			Required:      false,
			ClaimsMapping: map[string]string{"sub": "user_id"},
		}
		middleware := NewBearerJWKSMiddleware(config)
		defer middleware.Close()

		// Wait for initialization to complete
		middleware.jwksClient.WaitForInitialization()

		if !middleware.IsHealthy() {
			t.Error("Expected middleware to be healthy with valid JWKS")
		}
	})
}
