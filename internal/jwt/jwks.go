package jwt

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWKSCache represents a cached JWKS with TTL
type JWKSCache struct {
	fetchedAt time.Time
	keysByID  map[string]*rsa.PublicKey
	ttl       time.Duration
}

// JWKSResponse represents the JWKS response structure
type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	KeyType       string   `json:"kty"`
	KeyID         string   `json:"kid"`
	Usage         string   `json:"use"`
	X509CertChain []string `json:"x5c"`
	Modulus       string   `json:"n"`
	Exponent      string   `json:"e"`
}

// JWKSClient manages JWKS fetching and caching
type JWKSClient struct {
	jwksURL     string
	cache       *JWKSCache
	cacheMutex  sync.RWMutex
	httpClient  *http.Client
	fallbackTTL time.Duration

	// Background goroutine management
	ctx         context.Context
	cancel      context.CancelFunc
	refreshDone chan struct{}
	initialized chan struct{} // signals when initial fetch is complete

	// Exponential backoff for failed refresh attempts
	failureCount int
}

// NewJWKSClient creates a new JWKS client with configurable fallback TTL
func NewJWKSClient(jwksURL string, fallbackTTL time.Duration) *JWKSClient {
	ctx, cancel := context.WithCancel(context.Background())

	client := &JWKSClient{
		jwksURL: jwksURL,
		cache: &JWKSCache{
			keysByID: make(map[string]*rsa.PublicKey),
			ttl:      fallbackTTL,
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		fallbackTTL:  fallbackTTL,
		ctx:          ctx,
		cancel:       cancel,
		refreshDone:  make(chan struct{}),
		initialized:  make(chan struct{}),
		failureCount: 0,
	}

	// Start background refresh goroutine
	go client.backgroundRefresh()

	return client
}

// WaitForInitialization waits for the initial JWKS fetch to complete (mainly for testing)
func (c *JWKSClient) WaitForInitialization() {
	<-c.initialized
}

// Close stops the background refresh goroutine and cleans up resources
func (c *JWKSClient) Close() {
	c.cancel()
	<-c.refreshDone
}

// GetPublicKey retrieves the public key for the given key ID from local cache only
func (c *JWKSClient) GetPublicKey(kid string) (*rsa.PublicKey, error) {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	rsaPublicKey, exists := c.cache.keysByID[kid]
	if !exists {
		return nil, fmt.Errorf("key not found for kid: %s", kid)
	}

	return rsaPublicKey, nil
}

// backgroundRefresh runs in a background goroutine to proactively refresh JWKS before expiration
func (c *JWKSClient) backgroundRefresh() {
	defer close(c.refreshDone)

	// Initial fetch to populate cache
	initialFetchDone := false
	c.performRefresh()
	if !initialFetchDone {
		close(c.initialized)
		initialFetchDone = true
	}

	for {
		// Calculate next refresh time
		c.cacheMutex.RLock()
		var waitDuration time.Duration

		if len(c.cache.keysByID) > 0 && c.failureCount == 0 {
			// We have valid cache and no recent failures, calculate next refresh time (80% of TTL)
			refreshTime := c.cache.fetchedAt.Add(time.Duration(float64(c.cache.ttl) * 0.8))
			waitDuration = time.Until(refreshTime)
			if waitDuration < 0 {
				waitDuration = 0
			}
		} else {
			// No valid cache or we have recent failures, use exponential backoff with jitter
			waitDuration = c.calculateBackoffDuration()
		}

		c.cacheMutex.RUnlock()

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(waitDuration):
			c.performRefresh()
			if !initialFetchDone {
				close(c.initialized)
				initialFetchDone = true
			}
		}
	}
}

// calculateBackoffDuration calculates the wait duration with exponential backoff and jitter
func (c *JWKSClient) calculateBackoffDuration() time.Duration {
	// If there are no failures, no backoff is needed
	if c.failureCount == 0 {
		return 0
	}

	// Base retry interval starts at 30 seconds
	baseInterval := 30 * time.Second

	// Calculate exponential backoff: min(baseInterval * 2^(failureCount-1), maxInterval)
	// This gives us: 1st failure = 30s, 2nd failure = 60s, 3rd failure = 120s, etc.
	maxInterval := 10 * time.Minute // Cap backoff at 10 minutes
	backoffInterval := time.Duration(float64(baseInterval) * math.Pow(2, float64(c.failureCount-1)))
	if backoffInterval > maxInterval {
		backoffInterval = maxInterval
	}

	// Add jitter (±25% of the backoff interval) to prevent thundering herd
	jitterRange := float64(backoffInterval) * 0.25
	jitter := time.Duration(rand.Float64()*2*jitterRange - jitterRange)

	return backoffInterval + jitter
}

// performRefresh fetches JWKS and updates cache, handling errors gracefully
func (c *JWKSClient) performRefresh() {
	newCache, err := c.fetchJWKSFromServer()
	if err != nil {
		// Log error but don't block - this allows server to start without JWKS being available
		// In production, you might want to use a proper logger
		log.Printf("JWKS refresh failed: %v", err)

		// Increment failure count for exponential backoff
		c.cacheMutex.Lock()
		c.failureCount++
		c.cacheMutex.Unlock()
		return
	}

	// Success - reset failure count and update cache
	c.cacheMutex.Lock()
	c.failureCount = 0
	c.cache.fetchedAt = newCache.fetchedAt
	c.cache.keysByID = newCache.keysByID
	c.cache.ttl = newCache.ttl
	c.cacheMutex.Unlock()
}

// fetchJWKSFromServer fetches JWKS from the server and returns a new cache without updating the existing one
func (c *JWKSClient) fetchJWKSFromServer() (*JWKSCache, error) {
	response, err := c.httpClient.Get(c.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", c.jwksURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status: %d", response.StatusCode)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWKS response: %w", err)
	}

	// Parse Cache-Control header to determine TTL
	cacheTTL := c.parseCacheControl(response.Header.Get("Cache-Control"))

	// Parse JWKS format
	var jwksResponse JWKSResponse
	if err := json.Unmarshal(responseBody, &jwksResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS response: %w", err)
	}

	// Convert JWKS to key map
	keysByID := make(map[string]*rsa.PublicKey)
	for _, key := range jwksResponse.Keys {
		if key.KeyType == "RSA" {
			rsaKey, err := c.parseRSAKey(key)
			if err != nil {
				// Log warning but continue with other keys
				continue
			}
			keysByID[key.KeyID] = rsaKey
		}
	}

	// Return new cache without updating the existing one
	return &JWKSCache{
		fetchedAt: time.Now(),
		keysByID:  keysByID,
		ttl:       cacheTTL,
	}, nil
}

// parseRSAKey parses an RSA key from JWK format
func (c *JWKSClient) parseRSAKey(key JWK) (*rsa.PublicKey, error) {
	// Try X.509 certificate first
	if len(key.X509CertChain) > 0 {
		certPEM := fmt.Sprintf("-----BEGIN CERTIFICATE-----\n%s\n-----END CERTIFICATE-----", key.X509CertChain[0])
		return convertCertPEMToRSAPublicKey(certPEM)
	}

	// Try modulus and exponent
	if key.Modulus != "" && key.Exponent != "" {
		return constructRSAPublicKey(key.Modulus, key.Exponent)
	}

	return nil, fmt.Errorf("RSA key %s has neither x5c nor n/e fields", key.KeyID)
}

// constructRSAPublicKey creates an RSA public key from base64url-encoded modulus and exponent
func constructRSAPublicKey(modulusB64 string, exponentB64 string) (*rsa.PublicKey, error) {
	// Decode base64url-encoded modulus
	modulusBytes, err := base64.RawURLEncoding.DecodeString(modulusB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode base64url-encoded exponent
	exponentBytes, err := base64.RawURLEncoding.DecodeString(exponentB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert to big integers
	modulus := new(big.Int).SetBytes(modulusBytes)
	exponent := new(big.Int).SetBytes(exponentBytes)

	// Create RSA public key
	rsaPublicKey := &rsa.PublicKey{
		N: modulus,
		E: int(exponent.Int64()),
	}

	return rsaPublicKey, nil
}

// convertCertPEMToRSAPublicKey converts a certificate PEM string to RSA public key
func convertCertPEMToRSAPublicKey(certPEM string) (*rsa.PublicKey, error) {
	pemBlock, _ := pem.Decode([]byte(certPEM))
	if pemBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block from certificate")
	}

	certificate, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse x509 certificate: %w", err)
	}

	rsaPublicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("certificate does not contain RSA public key")
	}

	return rsaPublicKey, nil
}

// ValidateToken validates a JWT token using the JWKS client
func (c *JWKSClient) ValidateToken(tokenString string, expectedIssuer, expectedAudience string) (jwt.MapClaims, error) {
	// Parse token to get header for key ID
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid not found in token header")
		}

		// Get public key for this key ID
		publicKey, err := c.GetPublicKey(kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get public key: %w", err)
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Check if token is valid
	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to extract claims")
	}

	// Validate issuer if specified
	if expectedIssuer != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != expectedIssuer {
			return nil, fmt.Errorf("invalid issuer: expected %s, got %v", expectedIssuer, claims["iss"])
		}
	}

	// Validate audience if specified
	if expectedAudience != "" {
		if aud, ok := claims["aud"].(string); !ok || aud != expectedAudience {
			return nil, fmt.Errorf("invalid audience: expected %s, got %v", expectedAudience, claims["aud"])
		}
	}

	return claims, nil
}

// parseCacheControl parses the Cache-Control header and returns TTL
func (c *JWKSClient) parseCacheControl(cacheControl string) time.Duration {
	if cacheControl == "" {
		// No cache control header - use fallback TTL
		return c.fallbackTTL
	}

	// Parse Cache-Control header directives
	directives := strings.Split(cacheControl, ",")
	for _, directive := range directives {
		directive = strings.TrimSpace(directive)

		// Check for max-age directive
		if strings.HasPrefix(directive, "max-age=") {
			maxAgeStr := strings.TrimPrefix(directive, "max-age=")
			if maxAge, err := strconv.Atoi(maxAgeStr); err == nil && maxAge >= 0 {
				return time.Duration(maxAge) * time.Second
			}
		}

		// Check for no-cache or no-store (treat as immediate expiry)
		if directive == "no-cache" || directive == "no-store" {
			return 0
		}
	}

	// If Cache-Control header exists but no max-age found, use fallback TTL
	return c.fallbackTTL
}
