package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPHeaderMiddleware(t *testing.T) {
	// Test basic functionality
	t.Run("ExtractsHeaderValue", func(t *testing.T) {
		config := HTTPHeaderConfig{
			Header:    "X-User-ID",
			Parameter: "user_id",
			Required:  false,
		}
		middleware := NewHTTPHeaderMiddleware(config)

		// Create a test handler that checks the context
		var capturedParams map[string]interface{}
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			capturedParams = GetMiddlewareParams(r)
			w.WriteHeader(http.StatusOK)
		}

		// Wrap the handler with middleware
		wrappedHandler := middleware.Wrap(testHandler)

		// Create a request with the header
		req, _ := http.NewRequest("POST", "/", nil)
		req.Header.Set("X-User-ID", "123")
		rr := httptest.NewRecorder()

		// Call the wrapped handler
		wrappedHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %v", rr.Code)
		}

		if capturedParams["user_id"] != "123" {
			t.Errorf("Expected user_id=123, got %v", capturedParams["user_id"])
		}
	})

	// Test missing required header
	t.Run("RequiredHeaderMissing", func(t *testing.T) {
		config := HTTPHeaderConfig{
			Header:    "X-User-ID",
			Parameter: "user_id",
			Required:  true,
		}
		middleware := NewHTTPHeaderMiddleware(config)

		testHandler := func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called when required header is missing")
		}

		wrappedHandler := middleware.Wrap(testHandler)

		req, _ := http.NewRequest("POST", "/", nil)
		rr := httptest.NewRecorder()

		wrappedHandler(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status BadRequest, got %v", rr.Code)
		}
	})

	// Test optional header missing
	t.Run("OptionalHeaderMissing", func(t *testing.T) {
		config := HTTPHeaderConfig{
			Header:    "X-User-ID",
			Parameter: "user_id",
			Required:  false,
		}
		middleware := NewHTTPHeaderMiddleware(config)

		var capturedParams map[string]interface{}
		testHandler := func(w http.ResponseWriter, r *http.Request) {
			capturedParams = GetMiddlewareParams(r)
			w.WriteHeader(http.StatusOK)
		}

		wrappedHandler := middleware.Wrap(testHandler)

		req, _ := http.NewRequest("POST", "/", nil)
		rr := httptest.NewRecorder()

		wrappedHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %v", rr.Code)
		}

		if _, exists := capturedParams["user_id"]; exists {
			t.Errorf("Expected user_id parameter not to be added when header is missing")
		}
	})
}

func TestMiddlewareChain(t *testing.T) {
	// Create two middleware instances
	config1 := HTTPHeaderConfig{
		Header:    "X-User-ID",
		Parameter: "user_id",
		Required:  false,
	}
	middleware1 := NewHTTPHeaderMiddleware(config1)

	config2 := HTTPHeaderConfig{
		Header:    "X-Tenant-ID",
		Parameter: "tenant_id",
		Required:  false,
	}
	middleware2 := NewHTTPHeaderMiddleware(config2)

	chain := Chain{middleware1, middleware2}

	var capturedParams map[string]interface{}
	testHandler := func(w http.ResponseWriter, r *http.Request) {
		capturedParams = GetMiddlewareParams(r)
		w.WriteHeader(http.StatusOK)
	}

	// Wrap handler with middleware chain
	wrappedHandler := chain.Wrap(testHandler)

	// Create request with both headers
	req, _ := http.NewRequest("POST", "/", nil)
	req.Header.Set("X-User-ID", "123")
	req.Header.Set("X-Tenant-ID", "tenant456")
	rr := httptest.NewRecorder()

	wrappedHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", rr.Code)
	}

	if capturedParams["user_id"] != "123" {
		t.Errorf("Expected user_id=123, got %v", capturedParams["user_id"])
	}

	if capturedParams["tenant_id"] != "tenant456" {
		t.Errorf("Expected tenant_id=tenant456, got %v", capturedParams["tenant_id"])
	}
}
