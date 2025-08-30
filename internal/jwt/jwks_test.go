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

	// Test getting a key
	key, err := client.GetPublicKey("test-key-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if key == nil {
		t.Fatal("Expected key, got nil")
	}

	// Test getting non-existent key
	_, err = client.GetPublicKey("non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent key")
	}
}

func TestJWKSClient_Cache(t *testing.T) {
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
		w.Header().Set("Cache-Control", "max-age=0")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJWKS))
	}))
	defer server.Close()

	// Use short TTL for testing (fallback when cache-control parsing fails)
	client := NewJWKSClient(server.URL, 100*time.Millisecond)

	// First request should fetch from server
	_, err := client.GetPublicKey("test-key-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("Expected 1 request, got %d", requestCount)
	}

	// Second request should fetch again due to max-age=0
	_, err = client.GetPublicKey("test-key-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Add a small delay to ensure the request is processed
	time.Sleep(10 * time.Millisecond)

	if requestCount != 2 {
		t.Fatalf("Expected 2 requests (cache expired due to max-age=0), got %d", requestCount)
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

	_, err := client.GetPublicKey("test-key-1")
	if err == nil {
		t.Fatal("Expected error for server error")
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

	_, err := client.GetPublicKey("test-key-1")
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestJWKSClient_CacheControlHeaders(t *testing.T) {
	tests := []struct {
		name                   string
		cacheControl           string
		expectImmediateRefetch bool
		waitForExpiration      bool
		waitDuration           time.Duration
	}{
		{
			name:                   "No cache control header - uses fallback TTL",
			cacheControl:           "",
			expectImmediateRefetch: false,
			waitForExpiration:      true,
			waitDuration:           1100 * time.Millisecond, // Wait longer than fallback TTL
		},
		{
			name:                   "max-age=0 - expires immediately",
			cacheControl:           "max-age=0",
			expectImmediateRefetch: true,
			waitForExpiration:      false,
		},
		{
			name:                   "no-cache - expires immediately",
			cacheControl:           "no-cache",
			expectImmediateRefetch: true,
			waitForExpiration:      false,
		},
		{
			name:                   "no-store - expires immediately",
			cacheControl:           "no-store",
			expectImmediateRefetch: true,
			waitForExpiration:      false,
		},
		{
			name:                   "max-age=1 - expires after 1 second",
			cacheControl:           "max-age=1",
			expectImmediateRefetch: false,
			waitForExpiration:      true,
			waitDuration:           1100 * time.Millisecond, // Wait longer than max-age
		},
		{
			name:                   "multiple directives with max-age=2",
			cacheControl:           "public, max-age=2, must-revalidate",
			expectImmediateRefetch: false,
			waitForExpiration:      true,
			waitDuration:           2100 * time.Millisecond, // Wait longer than max-age
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				if tt.cacheControl != "" {
					w.Header().Set("Cache-Control", tt.cacheControl)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(mockJWKS))
			}))
			defer server.Close()

			client := NewJWKSClient(server.URL, 1*time.Second)

			// First request
			_, err := client.GetPublicKey("test-key-1")
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if requestCount != 1 {
				t.Fatalf("Expected 1 request after first call, got %d", requestCount)
			}

			// Second request - test immediate behavior
			_, err = client.GetPublicKey("test-key-1")
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if tt.expectImmediateRefetch {
				// Should refetch immediately
				if requestCount != 2 {
					t.Fatalf("Expected 2 requests after immediate refetch test, got %d", requestCount)
				}
			} else {
				// Should use cache
				if requestCount != 1 {
					t.Fatalf("Expected 1 request after cache hit test, got %d", requestCount)
				}
			}

			// Test expiration behavior if specified
			if tt.waitForExpiration {
				// Wait for cache to expire
				time.Sleep(tt.waitDuration)

				// Should refetch after expiration
				_, err = client.GetPublicKey("test-key-1")
				if err != nil {
					t.Fatalf("Expected no error after expiration, got %v", err)
				}

				expectedRequestsAfterExpiration := 2
				if tt.expectImmediateRefetch {
					expectedRequestsAfterExpiration = 3 // Already refetched once
				}

				if requestCount != expectedRequestsAfterExpiration {
					t.Fatalf("Expected %d requests after expiration test, got %d", expectedRequestsAfterExpiration, requestCount)
				}
			}
		})
	}
}

func TestJWKSClient_RefetchForUnknownKey(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var mockJWKS string

		if requestCount == 1 {
			// First request - only test-key-1
			mockJWKS = `{
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
		} else {
			// Second request - add test-key-2
			mockJWKS = `{
				"keys": [
					{
						"kty": "RSA",
						"kid": "test-key-1",
						"use": "sig",
						"n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISzIWzYr_W6UU9dwuW6TU0DjW0nQcaOLGOjQhGnOGKZ9CW7PDNE2J",
						"e": "AQAB"
					},
					{
						"kty": "RSA",
						"kid": "test-key-2",
						"use": "sig",
						"n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISzIWzYr_W6UU9dwuW6TU0DjW0nQcaOLGOjQhGnOGKZ9CW7PDNE2J",
						"e": "AQAB"
					}
				]
			}`
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJWKS))
	}))
	defer server.Close()

	client := NewJWKSClient(server.URL, 10*time.Minute)

	// First request - get existing key
	_, err := client.GetPublicKey("test-key-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("Expected 1 request, got %d", requestCount)
	}

	// Request for unknown key - should trigger refetch
	_, err = client.GetPublicKey("test-key-2")
	if err != nil {
		t.Fatalf("Expected no error after refetch, got %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("Expected 2 requests (refetch for unknown key), got %d", requestCount)
	}
}

func TestJWKSClient_RefetchRateLimit(t *testing.T) {
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
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJWKS))
	}))
	defer server.Close()

	client := NewJWKSClient(server.URL, 10*time.Minute)
	// Set short refetch interval for testing
	client.refetchMinInterval = 100 * time.Millisecond

	// Initial request to populate cache
	_, err := client.GetPublicKey("test-key-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("Expected 1 request, got %d", requestCount)
	}

	// Request for unknown key - should trigger refetch
	_, err = client.GetPublicKey("unknown-key")
	if err == nil {
		t.Fatal("Expected error for unknown key")
	}
	if requestCount != 2 {
		t.Fatalf("Expected 2 requests (refetch attempted), got %d", requestCount)
	}

	// Immediate second request for unknown key - should NOT trigger refetch due to rate limit
	_, err = client.GetPublicKey("another-unknown-key")
	if err == nil {
		t.Fatal("Expected error for unknown key")
	}
	if requestCount != 2 {
		t.Fatalf("Expected 2 requests (rate limited), got %d", requestCount)
	}

	// Wait for rate limit to pass
	time.Sleep(150 * time.Millisecond)

	// Third request for unknown key - should trigger refetch again
	_, err = client.GetPublicKey("yet-another-unknown-key")
	if err == nil {
		t.Fatal("Expected error for unknown key")
	}
	if requestCount != 3 {
		t.Fatalf("Expected 3 requests (rate limit passed), got %d", requestCount)
	}
}
