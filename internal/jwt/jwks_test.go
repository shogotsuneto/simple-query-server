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
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJWKS))
	}))
	defer server.Close()

	// Use short TTL for testing
	client := NewJWKSClient(server.URL, 100*time.Millisecond)

	// First request should fetch from server
	_, err := client.GetPublicKey("test-key-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("Expected 1 request, got %d", requestCount)
	}

	// Second request should use cache
	_, err = client.GetPublicKey("test-key-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("Expected 1 request (cached), got %d", requestCount)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third request should fetch from server again
	_, err = client.GetPublicKey("test-key-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("Expected 2 requests (cache expired), got %d", requestCount)
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
