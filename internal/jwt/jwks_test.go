package jwt

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestJWKSClient_GetPublicKey(t *testing.T) {
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
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJWKS))
	}))
	defer server.Close()

	client := NewJWKSClient(server.URL, 10*time.Minute)
	defer client.Close()

	// Wait for initial fetch to complete
	client.WaitForInitialization()

	// Test getting a key
	key, err := client.GetPublicKey("test-key-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if key == nil {
		t.Fatal("Expected key, got nil")
	}

	// Test getting non-existent key - should fail immediately (no request-triggered refetch)
	_, err = client.GetPublicKey("non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent key")
	}
}

func TestJWKSClient_BackgroundRefresh(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
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
		w.Header().Set("Content-Type", "application/json")
		// Set a short max-age for testing
		w.Header().Set("Cache-Control", "max-age=1")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJWKS))
	}))
	defer server.Close()

	client := NewJWKSClient(server.URL, 100*time.Millisecond)
	defer client.Close()

	// Wait for initial fetch
	client.WaitForInitialization()

	// Should have made initial request
	if requestCount != 1 {
		t.Fatalf("Expected 1 initial request, got %d", requestCount)
	}

	// Multiple GetPublicKey calls should not trigger additional requests
	for i := 0; i < 5; i++ {
		_, err := client.GetPublicKey("test-key-1")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	}

	// Should still be 1 request (no request-triggered refetch)
	if requestCount != 1 {
		t.Fatalf("Expected 1 request after multiple gets, got %d", requestCount)
	}

	// Wait for background refresh (at 80% of 1 second = 800ms)
	time.Sleep(900 * time.Millisecond)

	// Should have refreshed in background
	if requestCount < 2 {
		t.Fatalf("Expected at least 2 requests after background refresh, got %d", requestCount)
	}
}

func TestConstructRSAPublicKey(t *testing.T) {
	// Test with valid modulus and exponent
	modulus := "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISzIWzYr_W6UU9dwuW6TU0DjW0nQcaOLGOjQhGnOGKZ9CW7PDNE2J"
	exponent := "AQAB"

	key, err := constructRSAPublicKey(modulus, exponent)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if key == nil {
		t.Fatal("Expected key, got nil")
	}
	if key.E != 65537 { // AQAB is 65537 in decimal
		t.Fatalf("Expected exponent 65537, got %d", key.E)
	}
}

func TestJWKSClient_ErrorHandling(t *testing.T) {
	// Test with server returning error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewJWKSClient(server.URL, 10*time.Minute)
	defer client.Close()

	// Wait for initial fetch attempt (will fail)
	client.WaitForInitialization()

	// Should get error for any key since cache is empty
	_, err := client.GetPublicKey("test-key-1")
	if err == nil {
		t.Fatal("Expected error when server returns error and cache is empty")
	}
}

func TestJWKSClient_InvalidJSON(t *testing.T) {
	// Test with invalid JSON response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewJWKSClient(server.URL, 10*time.Minute)
	defer client.Close()

	// Wait for initial fetch attempt (will fail)
	client.WaitForInitialization()

	// Should get error for any key since cache is empty
	_, err := client.GetPublicKey("test-key-1")
	if err == nil {
		t.Fatal("Expected error for invalid JSON when cache is empty")
	}
}

// TestJWKSClient_CacheControlHeaders tests cache control header parsing for background refresh timing
func TestJWKSClient_CacheControlHeaders(t *testing.T) {
	tests := []struct {
		name         string
		cacheControl string
		expectedTTL  time.Duration
	}{
		{
			name:         "No cache control header - uses fallback TTL",
			cacheControl: "",
			expectedTTL:  1 * time.Second, // fallback TTL
		},
		{
			name:         "max-age=2",
			cacheControl: "max-age=2",
			expectedTTL:  2 * time.Second,
		},
		{
			name:         "multiple directives with max-age=3",
			cacheControl: "public, max-age=3, must-revalidate",
			expectedTTL:  3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				w.Header().Set("Content-Type", "application/json")
				if tt.cacheControl != "" {
					w.Header().Set("Cache-Control", tt.cacheControl)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(mockJWKS))
			}))
			defer server.Close()

			client := NewJWKSClient(server.URL, 1*time.Second)
			defer client.Close()

			// Wait for initial fetch
			client.WaitForInitialization()

			// Check that the cache TTL was set correctly
			client.cacheMutex.RLock()
			actualTTL := client.cache.ttl
			client.cacheMutex.RUnlock()

			if actualTTL != tt.expectedTTL {
				t.Fatalf("Expected TTL %v, got %v", tt.expectedTTL, actualTTL)
			}

			// Test that key can be retrieved
			_, err := client.GetPublicKey("test-key-1")
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
		})
	}
}

func TestJWKSClient_NoRequestTriggeredRefetch(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Always return only test-key-1, never add new keys
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

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJWKS))
	}))
	defer server.Close()

	client := NewJWKSClient(server.URL, 10*time.Minute)
	defer client.Close()

	// Wait for initial fetch
	client.WaitForInitialization()

	// First request - get existing key
	_, err := client.GetPublicKey("test-key-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("Expected 1 request, got %d", requestCount)
	}

	// Request for unknown key - should NOT trigger refetch
	_, err = client.GetPublicKey("test-key-2")
	if err == nil {
		t.Fatal("Expected error for unknown key")
	}
	if requestCount != 1 {
		t.Fatalf("Expected 1 request (no refetch for unknown key), got %d", requestCount)
	}
}

// This test is removed as it was testing the old rate limiting behavior for request-triggered refetch
// The new implementation doesn't have request-triggered refetch, so this test is no longer relevant

func TestJWKSClient_BackoffCalculation(t *testing.T) {
	client := NewJWKSClient("http://example.com", 10*time.Minute)
	defer client.Close()

	tests := []struct {
		name         string
		failureCount int
		minExpected  time.Duration
		maxExpected  time.Duration
	}{
		{
			name:         "No failures",
			failureCount: 0,
			minExpected:  0, // Should return 0 immediately
			maxExpected:  0, // Should return 0 immediately
		},
		{
			name:         "One failure",
			failureCount: 1,
			minExpected:  22 * time.Second, // 30s - 25% jitter
			maxExpected:  38 * time.Second, // 30s + 25% jitter
		},
		{
			name:         "Two failures",
			failureCount: 2,
			minExpected:  45 * time.Second, // 60s - 25% jitter
			maxExpected:  75 * time.Second, // 60s + 25% jitter
		},
		{
			name:         "Three failures",
			failureCount: 3,
			minExpected:  90 * time.Second,  // 120s - 25% jitter
			maxExpected:  150 * time.Second, // 120s + 25% jitter
		},
		{
			name:         "Many failures (capped at max)",
			failureCount: 10,
			minExpected:  7*time.Minute + 30*time.Second,  // 10m - 25% jitter
			maxExpected:  12*time.Minute + 30*time.Second, // 10m + 25% jitter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client.cacheMutex.Lock()
			client.failureCount = tt.failureCount
			client.cacheMutex.Unlock()

			duration := client.calculateBackoffDuration()

			if tt.failureCount == 0 {
				// For no failures, we expect exactly 0
				if duration != 0 {
					t.Fatalf("Expected duration 0 for no failures, got %v", duration)
				}
			} else {
				// For failures, check the range with jitter
				if duration < tt.minExpected || duration > tt.maxExpected {
					t.Fatalf("Expected duration between %v and %v, got %v", tt.minExpected, tt.maxExpected, duration)
				}
			}
		})
	}
}

// TestJWKSClient_IsHealthy tests the health check functionality
func TestJWKSClient_IsHealthy(t *testing.T) {
	t.Run("HealthyWithValidCache", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "max-age=3600") // 1 hour TTL
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockJWKS))
		}))
		defer server.Close()

		client := NewJWKSClient(server.URL, 10*time.Minute)
		defer client.Close()

		// Before initialization, should be unhealthy
		if client.IsHealthy() {
			t.Fatal("Expected unhealthy before initialization")
		}

		// Wait for initialization
		client.WaitForInitialization()

		// After successful initialization, should be healthy
		if !client.IsHealthy() {
			t.Fatal("Expected healthy after initialization with valid JWKS")
		}
	})

	t.Run("UnhealthyWithNoKeys", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return JWKS with no keys
			mockJWKS := `{"keys": []}`
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockJWKS))
		}))
		defer server.Close()

		client := NewJWKSClient(server.URL, 10*time.Minute)
		defer client.Close()

		client.WaitForInitialization()

		// Should be unhealthy with no keys
		if client.IsHealthy() {
			t.Fatal("Expected unhealthy with no keys")
		}
	})

	t.Run("HealthyWithFailuresButValidCache", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount == 1 {
				// First request succeeds
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
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Cache-Control", "max-age=3600") // Long TTL for testing
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(mockJWKS))
			} else {
				// Subsequent requests fail
				w.WriteHeader(http.StatusInternalServerError)
			}
		}))
		defer server.Close()

		client := NewJWKSClient(server.URL, 1*time.Hour) // Long TTL
		defer client.Close()

		client.WaitForInitialization()

		// Should be healthy initially
		if !client.IsHealthy() {
			t.Fatal("Expected healthy after successful initialization")
		}

		// Simulate a failed refresh by manually incrementing failure count
		client.cacheMutex.Lock()
		client.failureCount = 1
		client.cacheMutex.Unlock()

		// Should still be healthy because cache is valid, even with failures
		// Failure count doesn't matter if cache is still valid - refetch is scheduled before expiry
		if !client.IsHealthy() {
			t.Fatal("Expected healthy with failure count > 0 but valid cache")
		}
	})

	t.Run("UnhealthyWithExpiredCache", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "max-age=1") // 1 second TTL
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockJWKS))
		}))
		defer server.Close()

		client := NewJWKSClient(server.URL, 1*time.Second)
		defer client.Close()

		client.WaitForInitialization()

		// Should be healthy initially
		if !client.IsHealthy() {
			t.Fatal("Expected healthy after initialization")
		}

		// Manually set an expired cache time (simulate expired cache without waiting)
		client.cacheMutex.Lock()
		client.cache.fetchedAt = time.Now().Add(-2 * time.Second) // 2 seconds ago
		client.cacheMutex.Unlock()

		// Should now be unhealthy due to expired cache
		if client.IsHealthy() {
			t.Fatal("Expected unhealthy with expired cache")
		}
	})
}
