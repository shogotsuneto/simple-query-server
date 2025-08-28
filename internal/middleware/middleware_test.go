package middleware

import (
	"net/http"
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

		// Create a request with the header
		req, _ := http.NewRequest("POST", "/", nil)
		req.Header.Set("X-User-ID", "123")

		params := map[string]interface{}{"existing": "value"}
		result, err := middleware.Process(req, params)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result["user_id"] != "123" {
			t.Errorf("Expected user_id=123, got %v", result["user_id"])
		}

		if result["existing"] != "value" {
			t.Errorf("Expected existing parameter to be preserved")
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

		req, _ := http.NewRequest("POST", "/", nil)
		params := map[string]interface{}{}

		_, err := middleware.Process(req, params)

		if err == nil {
			t.Fatal("Expected error for missing required header")
		}

		expected := "required header 'X-User-ID' is missing"
		if err.Error() != expected {
			t.Errorf("Expected error '%s', got '%s'", expected, err.Error())
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

		req, _ := http.NewRequest("POST", "/", nil)
		params := map[string]interface{}{"existing": "value"}

		result, err := middleware.Process(req, params)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if _, exists := result["user_id"]; exists {
			t.Errorf("Expected user_id parameter not to be added when header is missing")
		}

		if result["existing"] != "value" {
			t.Errorf("Expected existing parameter to be preserved")
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

	// Create request with both headers
	req, _ := http.NewRequest("POST", "/", nil)
	req.Header.Set("X-User-ID", "123")
	req.Header.Set("X-Tenant-ID", "tenant456")

	params := map[string]interface{}{"original": "param"}
	result, err := chain.Process(req, params)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result["user_id"] != "123" {
		t.Errorf("Expected user_id=123, got %v", result["user_id"])
	}

	if result["tenant_id"] != "tenant456" {
		t.Errorf("Expected tenant_id=tenant456, got %v", result["tenant_id"])
	}

	if result["original"] != "param" {
		t.Errorf("Expected original parameter to be preserved")
	}
}
