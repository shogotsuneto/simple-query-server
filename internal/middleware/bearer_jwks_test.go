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
