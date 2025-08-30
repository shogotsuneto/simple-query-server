package jwt

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestManualBackgroundRefreshBehavior demonstrates the new background refresh behavior
func TestManualBackgroundRefreshBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual test in short mode")
	}

	requestCount := 0
	
	// Create a mock JWKS server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		fmt.Printf("JWKS Request #%d received at %v\n", requestCount, time.Now().Format("15:04:05.000"))
		
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
		w.Header().Set("Cache-Control", "max-age=3") // 3 second TTL
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockJWKS))
	}))
	defer server.Close()

	fmt.Printf("Starting test at %v\n", time.Now().Format("15:04:05.000"))
	
	// Create JWKS client with background refresh
	client := NewJWKSClient(server.URL, 10*time.Minute)
	defer client.Close()

	// Wait for initial fetch
	fmt.Println("Waiting for initial fetch...")
	client.WaitForInitialization()
	fmt.Printf("Initial fetch completed at %v\n", time.Now().Format("15:04:05.000"))

	// Verify initial request was made
	if requestCount != 1 {
		t.Fatalf("Expected 1 initial request, got %d", requestCount)
	}

	// Test multiple requests - should not trigger additional fetches
	fmt.Println("\nTesting multiple GetPublicKey calls (should not trigger refetch):")
	for i := 0; i < 5; i++ {
		_, err := client.GetPublicKey("test-key-1")
		if err != nil {
			t.Fatalf("Error getting key: %v", err)
		} else {
			fmt.Printf("Request %d: Got key successfully (no network request)\n", i+1)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Should still be only 1 request (no request-triggered refetch)
	if requestCount != 1 {
		t.Fatalf("Expected 1 request after multiple gets, got %d", requestCount)
	}

	// Test unknown key request - should not trigger refetch
	fmt.Println("\nTesting unknown key request (should not trigger refetch):")
	_, err := client.GetPublicKey("unknown-key")
	if err == nil {
		t.Fatal("Expected error for unknown key")
	}
	fmt.Printf("Unknown key error (expected): %v\n", err)

	// Should still be only 1 request (no request-triggered refetch for unknown keys)
	if requestCount != 1 {
		t.Fatalf("Expected 1 request after unknown key, got %d", requestCount)
	}

	// Wait for background refresh (should happen at 80% of 3 seconds = 2.4 seconds)
	fmt.Println("\nWaiting for background refresh (should happen around 2.4 seconds after initial fetch)...")
	time.Sleep(3 * time.Second)
	
	fmt.Printf("Test completed at %v\n", time.Now().Format("15:04:05.000"))
	fmt.Printf("Total JWKS requests: %d (expected: initial + background refresh)\n", requestCount)
	
	// Should have at least 2 requests now (initial + background refresh)
	if requestCount < 2 {
		t.Fatalf("Expected at least 2 requests (initial + background refresh), got %d", requestCount)
	}
	
	fmt.Println("\n✓ Test passed: Background refresh working, no request-triggered refetch")
}