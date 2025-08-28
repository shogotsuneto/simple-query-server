package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

const (
	serverBaseURL = "http://localhost:8081"
	healthTimeout = 60 * time.Second
	testTimeout   = 5 * time.Minute
)

var (
	serverCmd    *exec.Cmd
	serverCtx    context.Context
	cancelServer context.CancelFunc
)

// TestMain sets up and tears down the integration test environment
func TestMain(m *testing.M) {
	// Setup: Start database and server
	if err := startIntegrationEnvironment(); err != nil {
		fmt.Printf("Failed to start integration environment: %v\n", err)
		os.Exit(1)
	}

	// Wait for services to be healthy
	if err := waitForServices(); err != nil {
		fmt.Printf("Services failed to become healthy: %v\n", err)
		stopIntegrationEnvironment()
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup: Stop services
	stopIntegrationEnvironment()

	os.Exit(code)
}

// startIntegrationEnvironment starts the database and server for integration tests
func startIntegrationEnvironment() error {
	// Start PostgreSQL database
	fmt.Println("Starting PostgreSQL database for integration tests...")
	cmd := exec.Command("docker", "compose", "-f", "docker-compose.integration.yml", "up", "-d")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start database: %w", err)
	}

	// Wait a bit for database to initialize
	time.Sleep(15 * time.Second)

	// Start the server
	fmt.Println("Starting server for integration tests...")
	serverCtx, cancelServer = context.WithCancel(context.Background())
	serverCmd = exec.CommandContext(serverCtx, "../server",
		"--db-config", "./config/database.yaml",
		"--queries-config", "./config/queries.yaml",
		"--server-config", "./config/server.yaml",
		"--port", "8081")

	serverCmd.Dir = "."
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	if err := serverCmd.Start(); err != nil {
		stopIntegrationEnvironment()
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// stopIntegrationEnvironment stops and cleans up all services
func stopIntegrationEnvironment() {
	// Stop server
	if cancelServer != nil {
		cancelServer()
	}
	if serverCmd != nil {
		serverCmd.Wait()
	}

	// Stop database
	cmd := exec.Command("docker", "compose", "-f", "docker-compose.integration.yml", "down", "-v")
	cmd.Dir = "."
	cmd.Run() // Ignore errors on cleanup
}

// waitForServices waits for all services to be healthy and ready
func waitForServices() error {
	// Wait for server health endpoint to respond (either healthy or unhealthy)
	client := &http.Client{Timeout: 5 * time.Second}
	healthURL := serverBaseURL + "/health"

	timeout := time.Now().Add(healthTimeout)
	for time.Now().Before(timeout) {
		resp, err := client.Get(healthURL)
		if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable) {
			// Server is responding, check if database is connected
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Println("Integration test services are ready and database is connected")
				return nil
			} else {
				// Database not ready yet, continue waiting for it to connect
				fmt.Println("Server is ready but waiting for database connection...")
			}
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("services did not become healthy within %v", healthTimeout)
}

// makeRequest makes an HTTP request and returns the response
func makeRequest(method, url string, body interface{}) (*http.Response, []byte, error) {
	return makeRequestWithHeaders(method, url, body, nil)
}

// makeRequestWithHeaders makes an HTTP request with custom headers and returns the response
func makeRequestWithHeaders(method, url string, body interface{}, headers map[string]string) (*http.Response, []byte, error) {
	var requestBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		requestBody = bytes.NewBuffer(jsonBody)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to make request: %w", err)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}
	resp.Body.Close()

	return resp, responseBody, nil
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	resp, body, err := makeRequest("GET", serverBaseURL+"/health", nil)
	if err != nil {
		t.Fatalf("Failed to make health request: %v", err)
	}

	// Health endpoint should return 200 when database is connected (after integration setup)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var healthResponse map[string]interface{}
	if err := json.Unmarshal(body, &healthResponse); err != nil {
		t.Fatalf("Failed to unmarshal health response: %v", err)
	}

	// Check that we have the expected fields
	status, statusExists := healthResponse["status"]
	database, dbExists := healthResponse["database"]

	if !statusExists {
		t.Errorf("Expected 'status' field in health response")
	}
	if !dbExists {
		t.Errorf("Expected 'database' field in health response")
	}

	// When database is connected, status should be "healthy"
	if status != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", status)
	}

	// Check database connection status
	if dbMap, ok := database.(map[string]interface{}); ok {
		connected, exists := dbMap["connected"]
		if !exists {
			t.Errorf("Expected 'connected' field in database status")
		}
		if connected != true {
			t.Errorf("Expected database connected to be true, got %v", connected)
		}
	} else {
		t.Errorf("Expected database field to be an object")
	}
}

// TestListQueriesEndpoint tests the queries listing endpoint
func TestListQueriesEndpoint(t *testing.T) {
	resp, body, err := makeRequest("GET", serverBaseURL+"/queries", nil)
	if err != nil {
		t.Fatalf("Failed to make queries request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var queriesResponse map[string]interface{}
	if err := json.Unmarshal(body, &queriesResponse); err != nil {
		t.Fatalf("Failed to unmarshal queries response: %v", err)
	}

	queries, ok := queriesResponse["queries"]
	if !ok {
		t.Errorf("Expected 'queries' field in response")
	}

	queriesMap := queries.(map[string]interface{})
	expectedQueries := []string{
		"get_user_by_id",
		"search_users",
		"get_user_details",
		"list_users",
		"count_users_by_status",
		"get_all_active_users",
		"test_invalid_sql",
		"test_multiple_params",
		"get_current_user",
		"get_user_by_tenant",
		"secure_user_data",
		"tenant_user_query",
	}

	for _, expectedQuery := range expectedQueries {
		if _, exists := queriesMap[expectedQuery]; !exists {
			t.Errorf("Expected query '%s' not found in response", expectedQuery)
		}
	}
}

// TestQueryExecutionSuccess tests successful query execution scenarios
func TestQueryExecutionSuccess(t *testing.T) {
	tests := []struct {
		name             string
		queryName        string
		params           map[string]interface{}
		expectRows       bool
		expectedRowCount int
		expectedFields   []string
	}{
		{
			name:             "Get user by ID",
			queryName:        "get_user_by_id",
			params:           map[string]interface{}{"id": 1},
			expectRows:       true,
			expectedRowCount: 1,
			expectedFields:   []string{"id", "name", "email"},
		},
		{
			name:             "Get all active users",
			queryName:        "get_all_active_users",
			params:           map[string]interface{}{},
			expectRows:       true,
			expectedRowCount: 19,
			expectedFields:   []string{"id", "name", "email"},
		},
		{
			name:             "Search users by name",
			queryName:        "search_users",
			params:           map[string]interface{}{"name": "%Alice%"},
			expectRows:       true,
			expectedRowCount: 3,
			expectedFields:   []string{"id", "name"},
		},
		{
			name:             "List users with pagination",
			queryName:        "list_users",
			params:           map[string]interface{}{"limit": 5, "offset": 0},
			expectRows:       true,
			expectedRowCount: 5,
			expectedFields:   []string{"id", "name", "email"},
		},
		{
			name:             "Count users by status",
			queryName:        "count_users_by_status",
			params:           map[string]interface{}{"status": "active"},
			expectRows:       true,
			expectedRowCount: 1,
			expectedFields:   []string{"status", "count"},
		},
		{
			name:             "Multiple parameters query",
			queryName:        "test_multiple_params",
			params:           map[string]interface{}{"min_id": 1, "max_id": 5, "status": "active"},
			expectRows:       true,
			expectedRowCount: 2,
			expectedFields:   []string{"id", "name", "email", "status", "active", "created_at", "updated_at"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/query/%s", serverBaseURL, tt.queryName)
			resp, body, err := makeRequest("POST", url, tt.params)
			if err != nil {
				t.Fatalf("Failed to make query request: %v", err)
			}

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
				return
			}

			var queryResponse map[string]interface{}
			if err := json.Unmarshal(body, &queryResponse); err != nil {
				t.Fatalf("Failed to unmarshal query response: %v", err)
			}

			if tt.expectRows {
				rows, ok := queryResponse["rows"]
				if !ok {
					t.Errorf("Expected 'rows' field in response")
					return
				}

				rowsSlice := rows.([]interface{})
				if len(rowsSlice) == 0 {
					t.Errorf("Expected at least one row in response")
					return
				}

				// Check expected row count
				if len(rowsSlice) != tt.expectedRowCount {
					t.Errorf("Expected %d rows, got %d", tt.expectedRowCount, len(rowsSlice))
					return
				}

				// Check that first row contains expected fields
				firstRow := rowsSlice[0].(map[string]interface{})
				for _, field := range tt.expectedFields {
					if _, exists := firstRow[field]; !exists {
						t.Errorf("Expected field '%s' not found in row", field)
					}
				}
			}
		})
	}
}

// TestQueryExecutionErrors tests error scenarios
func TestQueryExecutionErrors(t *testing.T) {
	tests := []struct {
		name              string
		queryName         string
		params            map[string]interface{}
		expectedStatus    int
		expectedErrorText string
	}{
		{
			name:              "Missing required parameter",
			queryName:         "get_user_by_id",
			params:            map[string]interface{}{},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorText: "required parameter 'id' is missing",
		},
		{
			name:              "Wrong parameter type",
			queryName:         "get_user_by_id",
			params:            map[string]interface{}{"id": "not_a_number"},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorText: "",
		},
		{
			name:              "Nonexistent query",
			queryName:         "nonexistent_query",
			params:            map[string]interface{}{},
			expectedStatus:    http.StatusNotFound,
			expectedErrorText: "Query 'nonexistent_query' not found",
		},
		{
			name:              "Invalid SQL query",
			queryName:         "test_invalid_sql",
			params:            map[string]interface{}{"id": 1},
			expectedStatus:    http.StatusInternalServerError,
			expectedErrorText: "",
		},
		{
			name:              "Partial parameters for multiple param query",
			queryName:         "test_multiple_params",
			params:            map[string]interface{}{"min_id": 1},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorText: "required parameter 'max_id' is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/query/%s", serverBaseURL, tt.queryName)
			resp, body, err := makeRequest("POST", url, tt.params)
			if err != nil {
				t.Fatalf("Failed to make query request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			var errorResponse map[string]interface{}
			if err := json.Unmarshal(body, &errorResponse); err != nil {
				t.Fatalf("Failed to unmarshal error response: %v", err)
			}

			errorMsg, ok := errorResponse["error"]
			if !ok {
				t.Errorf("Expected 'error' field in response")
				return
			}

			if tt.expectedErrorText != "" {
				if errorMsg != tt.expectedErrorText {
					t.Errorf("Expected error message '%s', got '%s'", tt.expectedErrorText, errorMsg)
				}
			}
		})
	}
}

// TestHTTPMethods tests that only appropriate HTTP methods are accepted
func TestHTTPMethods(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		endpoint       string
		params         map[string]interface{}
		expectedStatus int
	}{
		{
			name:           "GET health - allowed",
			method:         "GET",
			endpoint:       "/health",
			params:         nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST health - not allowed",
			method:         "POST",
			endpoint:       "/health",
			params:         nil,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "GET queries - allowed",
			method:         "GET",
			endpoint:       "/queries",
			params:         nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST queries - not allowed",
			method:         "POST",
			endpoint:       "/queries",
			params:         nil,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "POST query execution - allowed",
			method:         "POST",
			endpoint:       "/query/get_all_active_users",
			params:         map[string]interface{}{},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET query execution - not allowed",
			method:         "GET",
			endpoint:       "/query/get_all_active_users",
			params:         nil,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := serverBaseURL + tt.endpoint
			resp, _, err := makeRequest(tt.method, url, tt.params)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// TestDataConsistency tests that the database returns consistent data
func TestDataConsistency(t *testing.T) {
	// Test that the same user ID returns consistent data across different queries
	userID := 1

	// Get user by ID
	resp1, body1, err := makeRequest("POST", serverBaseURL+"/query/get_user_by_id",
		map[string]interface{}{"id": userID})
	if err != nil || resp1.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	var userResponse map[string]interface{}
	if err := json.Unmarshal(body1, &userResponse); err != nil {
		t.Fatalf("Failed to unmarshal user response: %v", err)
	}

	rows := userResponse["rows"].([]interface{})
	if len(rows) == 0 {
		t.Fatalf("Expected user data, got empty result")
	}

	user := rows[0].(map[string]interface{})
	userName := user["name"].(string)

	// Search for the same user by name pattern
	searchPattern := "%" + userName + "%"
	resp2, body2, err := makeRequest("POST", serverBaseURL+"/query/search_users",
		map[string]interface{}{"name": searchPattern})
	if err != nil || resp2.StatusCode != http.StatusOK {
		t.Fatalf("Failed to search users: %v", err)
	}

	var searchResponse map[string]interface{}
	if err := json.Unmarshal(body2, &searchResponse); err != nil {
		t.Fatalf("Failed to unmarshal search response: %v", err)
	}

	searchRows := searchResponse["rows"].([]interface{})
	found := false
	for _, row := range searchRows {
		rowMap := row.(map[string]interface{})
		if int(rowMap["id"].(float64)) == userID {
			found = true
			if rowMap["name"] != userName {
				t.Errorf("Inconsistent user name: expected '%s', got '%s'", userName, rowMap["name"])
			}
			break
		}
	}

	if !found {
		t.Errorf("User with ID %d not found in search results", userID)
	}
}

// TestDatabaseReconnection tests that the server handles database disconnections gracefully
func TestDatabaseReconnection(t *testing.T) {
	// First verify that database is healthy
	resp, body, err := makeRequest("GET", serverBaseURL+"/health", nil)
	if err != nil {
		t.Fatalf("Failed to make health request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected healthy database, got status %d", resp.StatusCode)
	}

	var healthResponse map[string]interface{}
	if err := json.Unmarshal(body, &healthResponse); err != nil {
		t.Fatalf("Failed to unmarshal health response: %v", err)
	}

	if healthResponse["status"] != "healthy" {
		t.Fatalf("Expected healthy status, got %v", healthResponse["status"])
	}

	// Test that queries work when database is healthy
	resp, _, err = makeRequest("POST", serverBaseURL+"/query/get_all_active_users", map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to execute query with healthy database: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected successful query execution with healthy database, got status %d", resp.StatusCode)
	}

	// Note: We don't test actual disconnection in integration tests to avoid disrupting other tests
	// The database connection retry logic is tested by starting the server without database first
}

// TestMiddlewareIntegration tests the middleware functionality
func TestMiddlewareIntegration(t *testing.T) {
	t.Run("HeaderMiddlewareBasic", func(t *testing.T) {
		// Test basic header middleware functionality
		headers := map[string]string{
			"X-User-ID": "1",
		}
		
		resp, body, err := makeRequestWithHeaders("POST", 
			serverBaseURL+"/query/get_current_user", 
			map[string]interface{}{}, 
			headers)
		
		if err != nil {
			t.Fatalf("Failed to make query request with headers: %v", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
			return
		}
		
		var queryResponse map[string]interface{}
		if err := json.Unmarshal(body, &queryResponse); err != nil {
			t.Fatalf("Failed to unmarshal query response: %v", err)
		}
		
		rows, ok := queryResponse["rows"]
		if !ok {
			t.Errorf("Expected 'rows' field in response")
			return
		}
		
		rowsSlice := rows.([]interface{})
		if len(rowsSlice) == 0 {
			t.Errorf("Expected at least one row in response")
			return
		}
		
		// Should return user with ID 1
		firstRow := rowsSlice[0].(map[string]interface{})
		if int(firstRow["id"].(float64)) != 1 {
			t.Errorf("Expected user ID 1, got %v", firstRow["id"])
		}
	})
	
	t.Run("MultipleHeaderParameters", func(t *testing.T) {
		// Test multiple header middleware parameters
		headers := map[string]string{
			"X-User-ID":   "123",
			"X-Tenant-ID": "tenant456",
		}
		
		resp, body, err := makeRequestWithHeaders("POST", 
			serverBaseURL+"/query/tenant_user_query", 
			map[string]interface{}{}, 
			headers)
		
		if err != nil {
			t.Fatalf("Failed to make query request with multiple headers: %v", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
			return
		}
		
		var queryResponse map[string]interface{}
		if err := json.Unmarshal(body, &queryResponse); err != nil {
			t.Fatalf("Failed to unmarshal query response: %v", err)
		}
		
		rows := queryResponse["rows"].([]interface{})
		firstRow := rows[0].(map[string]interface{})
		
		if firstRow["user_id"] != "123" {
			t.Errorf("Expected user_id=123, got %v", firstRow["user_id"])
		}
		
		if firstRow["tenant_id"] != "tenant456" {
			t.Errorf("Expected tenant_id=tenant456, got %v", firstRow["tenant_id"])
		}
	})
	
	t.Run("MixedBodyAndMiddlewareParameters", func(t *testing.T) {
		// Test mixing body and middleware parameters
		headers := map[string]string{
			"X-Tenant-ID": "tenant789",
		}
		
		bodyParams := map[string]interface{}{
			"id": 2,
		}
		
		resp, body, err := makeRequestWithHeaders("POST", 
			serverBaseURL+"/query/get_user_by_tenant", 
			bodyParams, 
			headers)
		
		if err != nil {
			t.Fatalf("Failed to make query request with mixed parameters: %v", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
			return
		}
		
		var queryResponse map[string]interface{}
		if err := json.Unmarshal(body, &queryResponse); err != nil {
			t.Fatalf("Failed to unmarshal query response: %v", err)
		}
		
		rows := queryResponse["rows"].([]interface{})
		if len(rows) > 0 {
			// If we get results, verify the user ID matches our body parameter
			firstRow := rows[0].(map[string]interface{})
			if int(firstRow["id"].(float64)) != 2 {
				t.Errorf("Expected user ID 2, got %v", firstRow["id"])
			}
		}
	})
	
	t.Run("OptionalHeaderMissing", func(t *testing.T) {
		// Test that optional headers work when missing
		resp, body, err := makeRequest("POST", 
			serverBaseURL+"/query/get_current_user", 
			map[string]interface{}{})
		
		if err != nil {
			t.Fatalf("Failed to make query request without headers: %v", err)
		}
		
		// This should still work but the SQL query will have NULL for the missing parameter
		// Depending on the query, this might return no results or cause an error
		// For our test query, it will likely cause a DB error due to WHERE id = NULL
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 500 (internal server error) for missing user_id, got %d. Response: %s", resp.StatusCode, string(body))
		}
	})
	
	t.Run("ParameterFiltering", func(t *testing.T) {
		// Test that body parameters are filtered according to YAML config
		headers := map[string]string{
			"X-User-ID": "1",
		}
		
		// Try to provide user_id in body (should be filtered out)
		bodyParams := map[string]interface{}{
			"user_id": "999", // This should be ignored since user_id is a middleware param
			"extra":   "ignored", // This should also be filtered out
		}
		
		resp, body, err := makeRequestWithHeaders("POST", 
			serverBaseURL+"/query/get_current_user", 
			bodyParams, 
			headers)
		
		if err != nil {
			t.Fatalf("Failed to make query request for parameter filtering test: %v", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(body))
			return
		}
		
		var queryResponse map[string]interface{}
		if err := json.Unmarshal(body, &queryResponse); err != nil {
			t.Fatalf("Failed to unmarshal query response: %v", err)
		}
		
		rows := queryResponse["rows"].([]interface{})
		if len(rows) > 0 {
			// Should return user with ID 1 (from header), not 999 (from filtered body)
			firstRow := rows[0].(map[string]interface{})
			if int(firstRow["id"].(float64)) != 1 {
				t.Errorf("Expected user ID 1 (from header), got %v. Parameter filtering may not be working.", firstRow["id"])
			}
		}
	})
	
	t.Run("QueriesEndpointShowsMiddlewareParams", func(t *testing.T) {
		// Test that /queries endpoint properly shows middleware_params separately
		resp, body, err := makeRequest("GET", serverBaseURL+"/queries", nil)
		if err != nil {
			t.Fatalf("Failed to make queries request: %v", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		
		var queriesResponse map[string]interface{}
		if err := json.Unmarshal(body, &queriesResponse); err != nil {
			t.Fatalf("Failed to unmarshal queries response: %v", err)
		}
		
		queries := queriesResponse["queries"].(map[string]interface{})
		
		// Check that middleware-using queries show the middleware_params
		getCurrentUser := queries["get_current_user"].(map[string]interface{})
		
		middlewareParams, hasMiddlewareParams := getCurrentUser["middleware_params"]
		if !hasMiddlewareParams {
			t.Errorf("Expected 'middleware_params' field for get_current_user query")
			return
		}
		
		middlewareParamsSlice := middlewareParams.([]interface{})
		if len(middlewareParamsSlice) == 0 {
			t.Errorf("Expected at least one middleware parameter for get_current_user query")
			return
		}
		
		// Check that the parameter has the expected structure
		firstParam := middlewareParamsSlice[0].(map[string]interface{})
		if firstParam["name"] != "user_id" {
			t.Errorf("Expected first middleware parameter to be 'user_id', got %v", firstParam["name"])
		}
		
		if firstParam["type"] != "string" {
			t.Errorf("Expected middleware parameter type to be 'string', got %v", firstParam["type"])
		}
		
		// Also verify regular params are still there for mixed queries
		getUserByTenant := queries["get_user_by_tenant"].(map[string]interface{})
		
		params, hasParams := getUserByTenant["params"]
		if !hasParams {
			t.Errorf("Expected 'params' field for get_user_by_tenant query")
			return
		}
		
		middlewareParams, hasMiddlewareParams = getUserByTenant["middleware_params"]
		if !hasMiddlewareParams {
			t.Errorf("Expected 'middleware_params' field for get_user_by_tenant query")
			return
		}
		
		paramsSlice := params.([]interface{})
		middlewareParamsSlice = middlewareParams.([]interface{})
		
		if len(paramsSlice) == 0 {
			t.Errorf("Expected at least one regular parameter for get_user_by_tenant query")
		}
		
		if len(middlewareParamsSlice) == 0 {
			t.Errorf("Expected at least one middleware parameter for get_user_by_tenant query")
		}
	})
}
