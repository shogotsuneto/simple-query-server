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
	"strings"
	"testing"
	"time"
)

const (
	jwtServerBaseURL = "http://localhost:8082"
	jwksAPIBaseURL   = "http://localhost:3000"
	jwtTestTimeout   = 10 * time.Minute
)

var (
	jwtServerCmd    *exec.Cmd
	jwtServerCtx    context.Context
	cancelJWTServer context.CancelFunc
)

// TestJWTIntegration runs comprehensive JWT/JWKS authentication tests
func TestJWTIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping JWT integration test in short mode")
	}

	// Setup JWT test environment
	if err := startJWTTestEnvironment(); err != nil {
		t.Fatalf("Failed to start JWT test environment: %v", err)
	}
	defer stopJWTTestEnvironment()

	// Wait for services to be healthy
	if err := waitForJWTServices(); err != nil {
		t.Fatalf("JWT services failed to become healthy: %v", err)
	}

	// Run JWT-specific tests
	t.Run("JWKSAPIHealth", testJWKSAPIHealth)
	t.Run("JWTServerHealth", testJWTServerHealth)
	t.Run("OptionalAuthWithoutToken", testOptionalAuthWithoutToken)
	t.Run("OptionalAuthWithValidToken", testOptionalAuthWithValidToken)
	t.Run("OptionalAuthWithInvalidToken", testOptionalAuthWithInvalidToken)
	t.Run("ClaimsMapping", testClaimsMapping)
	t.Run("MixedParametersJWT", testMixedParametersJWT)
	t.Run("RequiredAuthTests", testRequiredAuth)
}

func startJWTTestEnvironment() error {
	fmt.Println("Starting JWT integration test environment...")

	// Start PostgreSQL database and JWKS Mock API using docker-compose
	cmd := exec.Command("docker", "compose", "up", "-d", "postgres", "jwks-mock-api")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start PostgreSQL and JWKS Mock API: %w", err)
	}

	// Wait for services to initialize
	time.Sleep(15 * time.Second)

	// Start our server with JWT configuration
	jwtServerCtx, cancelJWTServer = context.WithTimeout(context.Background(), jwtTestTimeout)
	jwtServerCmd = exec.CommandContext(jwtServerCtx,
		"../server",
		"--db-config", "../example/database.yaml",
		"--queries-config", "./config/queries-with-jwt.yaml",
		"--server-config", "./config/server-with-jwt.yaml",
		"--port", "8082")
	jwtServerCmd.Stdout = os.Stdout
	jwtServerCmd.Stderr = os.Stderr

	if err := jwtServerCmd.Start(); err != nil {
		return fmt.Errorf("failed to start JWT server: %w", err)
	}

	return nil
}

func stopJWTTestEnvironment() {
	fmt.Println("Stopping JWT integration test environment...")

	if cancelJWTServer != nil {
		cancelJWTServer()
	}
	if jwtServerCmd != nil && jwtServerCmd.Process != nil {
		jwtServerCmd.Process.Kill()
		jwtServerCmd.Wait()
	}

	// Stop all services
	exec.Command("docker", "compose", "down").Run()
}

func waitForJWTServices() error {
	// Wait for JWKS API
	if err := waitForService(jwksAPIBaseURL+"/health", 30*time.Second); err != nil {
		return fmt.Errorf("JWKS API not healthy: %w", err)
	}

	// Wait for our JWT server
	if err := waitForService(jwtServerBaseURL+"/health", 60*time.Second); err != nil {
		return fmt.Errorf("JWT server not healthy: %w", err)
	}

	return nil
}

func waitForService(url string, timeout time.Duration) error {
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("service at %s not ready within %v", url, timeout)
}

func generateJWTToken(claims map[string]interface{}) (string, error) {
	requestBody := map[string]interface{}{
		"claims":    claims,
		"expiresIn": 3600,
	}

	bodyBytes, _ := json.Marshal(requestBody)
	resp, err := http.Post(jwksAPIBaseURL+"/generate-token", "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to generate token, status: %d", resp.StatusCode)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	token, ok := response["token"].(string)
	if !ok {
		return "", fmt.Errorf("token not found in response")
	}

	return token, nil
}

func makeJWTRequest(method, url string, headers map[string]string, body map[string]interface{}) (*http.Response, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return resp, nil, err
	}

	return resp, respBody, nil
}

func testJWKSAPIHealth(t *testing.T) {
	resp, err := http.Get(jwksAPIBaseURL + "/health")
	if err != nil {
		t.Fatalf("Failed to check JWKS API health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected JWKS API health check to return 200, got %d", resp.StatusCode)
	}
}

func testJWTServerHealth(t *testing.T) {
	resp, err := http.Get(jwtServerBaseURL + "/health")
	if err != nil {
		t.Fatalf("Failed to check JWT server health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected JWT server health check to return 200, got %d", resp.StatusCode)
	}
}

func testOptionalAuthWithoutToken(t *testing.T) {
	resp, body, err := makeJWTRequest("POST", jwtServerBaseURL+"/query/get_all_active_users", nil, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to make request without token: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected request without token to succeed (optional auth), got status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	rows, ok := result["rows"].([]interface{})
	if !ok || len(rows) == 0 {
		t.Errorf("Expected non-empty rows in response without token, got %v", result)
	}
}

func testOptionalAuthWithValidToken(t *testing.T) {
	token, err := generateJWTToken(map[string]interface{}{
		"sub":   "2",
		"role":  "user",
		"email": "user@example.com",
	})
	if err != nil {
		t.Fatalf("Failed to generate JWT token: %v", err)
	}

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	resp, body, err := makeJWTRequest("POST", jwtServerBaseURL+"/query/get_user_by_jwt_sub", headers, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to make request with valid token: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected request with valid token to succeed, got status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	rows, ok := result["rows"].([]interface{})
	if !ok || len(rows) != 1 {
		t.Errorf("Expected exactly 1 row in response with valid token, got %v", result)
	}

	if len(rows) > 0 {
		firstRow := rows[0].(map[string]interface{})
		if firstRow["id"] != float64(2) { // JSON numbers are parsed as float64
			t.Errorf("Expected user ID 2 from JWT sub claim, got %v", firstRow["id"])
		}
	}
}

func testOptionalAuthWithInvalidToken(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer invalid.jwt.token",
	}

	resp, body, err := makeJWTRequest("POST", jwtServerBaseURL+"/query/get_all_active_users", headers, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to make request with invalid token: %v", err)
	}

	// Should succeed because auth is optional
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected request with invalid token to succeed (optional auth), got status %d: %s", resp.StatusCode, string(body))
	}
}

func testClaimsMapping(t *testing.T) {
	token, err := generateJWTToken(map[string]interface{}{
		"sub":   "3",
		"role":  "admin",
		"email": "admin@test.com",
	})
	if err != nil {
		t.Fatalf("Failed to generate JWT token: %v", err)
	}

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	resp, body, err := makeJWTRequest("POST", jwtServerBaseURL+"/query/get_user_profile_with_claims", headers, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to make request for claims mapping test: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected claims mapping request to succeed, got status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	rows, ok := result["rows"].([]interface{})
	if !ok || len(rows) != 1 {
		t.Errorf("Expected exactly 1 row in claims mapping response, got %v", result)
	}

	if len(rows) > 0 {
		row := rows[0].(map[string]interface{})

		// Check that JWT claims were properly mapped
		if row["provided_role"] != "admin" {
			t.Errorf("Expected provided_role 'admin' from JWT role claim, got %v", row["provided_role"])
		}

		if row["provided_email"] != "admin@test.com" {
			t.Errorf("Expected provided_email 'admin@test.com' from JWT email claim, got %v", row["provided_email"])
		}

		if row["id"] != float64(3) {
			t.Errorf("Expected user ID 3 from JWT sub claim, got %v", row["id"])
		}
	}
}

func testMixedParametersJWT(t *testing.T) {
	token, err := generateJWTToken(map[string]interface{}{
		"sub":   "1",
		"role":  "user",
		"email": "user@example.com",
	})
	if err != nil {
		t.Fatalf("Failed to generate JWT token: %v", err)
	}

	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}

	// Test mixing JWT claims with request body parameters
	requestBody := map[string]interface{}{
		"search_term": "%Alice%",
	}

	resp, body, err := makeJWTRequest("POST", jwtServerBaseURL+"/query/search_with_auth", headers, requestBody)
	if err != nil {
		t.Fatalf("Failed to make mixed parameters request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected mixed parameters request to succeed, got status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	rows, ok := result["rows"].([]interface{})
	if !ok {
		t.Errorf("Expected rows in mixed parameters response, got %v", result)
	}

	// Should find Alice users and show authenticated_user from JWT
	if len(rows) > 0 {
		row := rows[0].(map[string]interface{})
		if row["authenticated_user"] != "1" {
			t.Errorf("Expected authenticated_user '1' from JWT sub claim, got %v", row["authenticated_user"])
		}
		
		// Verify that the name contains "Alice" as expected from search term
		name, ok := row["name"].(string)
		if !ok {
			t.Errorf("Expected name field to be a string, got %v", row["name"])
		} else if !strings.Contains(name, "Alice") {
			t.Errorf("Expected name to contain 'Alice', got %v", name)
		}
	}
}

func testRequiredAuth(t *testing.T) {
	// Start a server with required authentication
	serverCtx, cancelServer := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelServer()

	serverCmd := exec.CommandContext(serverCtx,
		"../server",
		"--db-config", "../example/database.yaml",
		"--queries-config", "./config/queries-with-jwt.yaml",
		"--server-config", "./config/server-required-auth.yaml",
		"--port", "8083")

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start required auth server: %v", err)
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
			serverCmd.Wait()
		}
	}()

	// Wait for server to start
	requiredAuthURL := "http://localhost:8083"
	if err := waitForService(requiredAuthURL+"/health", 30*time.Second); err != nil {
		t.Fatalf("Required auth server not ready: %v", err)
	}

	// Test 1: Request without token should fail
	resp, body, err := makeJWTRequest("POST", requiredAuthURL+"/query/get_all_active_users", nil, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to make request without token to required auth server: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized for request without token to required auth server, got %d: %s", resp.StatusCode, string(body))
	}

	// Test 2: Request with invalid token should fail
	headers := map[string]string{
		"Authorization": "Bearer invalid.token",
	}

	resp, body, err = makeJWTRequest("POST", requiredAuthURL+"/query/get_all_active_users", headers, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to make request with invalid token to required auth server: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized for invalid token to required auth server, got %d: %s", resp.StatusCode, string(body))
	}

	// Test 3: Request with valid token should succeed
	token, err := generateJWTToken(map[string]interface{}{
		"sub":   "5",
		"role":  "admin",
		"email": "admin@example.com",
	})
	if err != nil {
		t.Fatalf("Failed to generate JWT token for required auth test: %v", err)
	}

	headers["Authorization"] = "Bearer " + token

	resp, body, err = makeJWTRequest("POST", requiredAuthURL+"/query/get_user_by_jwt_sub", headers, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to make request with valid token to required auth server: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK for valid token to required auth server, got %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response from required auth server: %v", err)
	}

	rows, ok := result["rows"].([]interface{})
	if !ok || len(rows) != 1 {
		t.Errorf("Expected exactly 1 row from required auth server, got %v", result)
	}

	if len(rows) > 0 {
		row := rows[0].(map[string]interface{})
		if row["id"] != float64(5) { // JWT sub claim was "5"
			t.Errorf("Expected user ID 5 from JWT sub claim in required auth test, got %v", row["id"])
		}
	}
}
