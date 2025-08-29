package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWKSCache represents a cached JWKS with TTL and health status
type JWKSCache struct {
	fetchedAt    time.Time
	keysByID     map[string]*rsa.PublicKey
	ttl          time.Duration
	lastError    error
	lastFetchOK  bool
	refreshAfter time.Time // When to start background refresh attempts
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
	jwksURL            string
	cache              *JWKSCache
	cacheMutex         sync.RWMutex
	httpClient         *http.Client
	minRefreshInterval time.Duration // Minimum interval between refresh attempts
	lastRefreshAttempt time.Time
	refreshMutex       sync.Mutex
}

// NewJWKSClient creates a new JWKS client with configurable cache TTL and refresh interval
func NewJWKSClient(jwksURL string, cacheTTL time.Duration) *JWKSClient {
	return NewJWKSClientWithRefreshInterval(jwksURL, cacheTTL, 5*time.Minute)
}

// NewJWKSClientWithRefreshInterval creates a new JWKS client with configurable cache TTL and minimum refresh interval
func NewJWKSClientWithRefreshInterval(jwksURL string, cacheTTL, minRefreshInterval time.Duration) *JWKSClient {
	return &JWKSClient{
		jwksURL: jwksURL,
		cache: &JWKSCache{
			keysByID:    make(map[string]*rsa.PublicKey),
			ttl:         cacheTTL,
			lastFetchOK: false,
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		minRefreshInterval: minRefreshInterval,
	}
}

// GetPublicKey retrieves the public key for the given key ID
func (c *JWKSClient) GetPublicKey(kid string) (*rsa.PublicKey, error) {
	cache, err := c.fetchJWKSWithCache()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	rsaPublicKey, exists := cache.keysByID[kid]
	if !exists {
		// Try refreshing cache in case this is a new key
		cache, err = c.tryRefreshForUnknownKey(kid)
		if err != nil {
			return nil, fmt.Errorf("key not found for kid: %s", kid)
		}

		rsaPublicKey, exists = cache.keysByID[kid]
		if !exists {
			return nil, fmt.Errorf("key not found for kid: %s", kid)
		}
	}

	return rsaPublicKey, nil
}

// IsHealthy returns whether the JWKS client is healthy
func (c *JWKSClient) IsHealthy() bool {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	// Healthy if we have keys and the last fetch was OK
	// If we've never fetched, we're considered healthy
	if c.cache.fetchedAt.IsZero() {
		return true
	}

	return len(c.cache.keysByID) > 0 && c.cache.lastFetchOK
}

// GetLastError returns the last error encountered during JWKS fetching
func (c *JWKSClient) GetLastError() error {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()
	return c.cache.lastError
}

// fetchJWKSWithCache fetches JWKS with caching logic
func (c *JWKSClient) fetchJWKSWithCache() (*JWKSCache, error) {
	c.cacheMutex.RLock()
	// Check if cache is still valid
	if time.Since(c.cache.fetchedAt) < c.cache.ttl && c.cache.keysByID != nil && len(c.cache.keysByID) > 0 {
		// Check if we should start background refresh
		if time.Now().After(c.cache.refreshAfter) {
			c.cacheMutex.RUnlock()
			go c.backgroundRefresh()
			c.cacheMutex.RLock()
		}
		cache := c.cache
		c.cacheMutex.RUnlock()
		return cache, nil
	}

	// Check if we have existing keys that we should preserve on error
	hasExistingKeys := len(c.cache.keysByID) > 0
	c.cacheMutex.RUnlock()

	// Need to fetch fresh JWKS
	return c.fetchJWKS(hasExistingKeys)
}

// tryRefreshForUnknownKey attempts to refresh JWKS when an unknown key is requested
func (c *JWKSClient) tryRefreshForUnknownKey(kid string) (*JWKSCache, error) {
	c.refreshMutex.Lock()
	defer c.refreshMutex.Unlock()

	// Check if enough time has passed since last refresh attempt
	if time.Since(c.lastRefreshAttempt) < c.minRefreshInterval {
		c.cacheMutex.RLock()
		cache := c.cache
		c.cacheMutex.RUnlock()
		return cache, fmt.Errorf("rate limited")
	}

	c.lastRefreshAttempt = time.Now()
	return c.fetchJWKS(true)
}

// backgroundRefresh performs background refresh
func (c *JWKSClient) backgroundRefresh() {
	c.refreshMutex.Lock()
	defer c.refreshMutex.Unlock()

	// Check if enough time has passed since last refresh attempt
	if time.Since(c.lastRefreshAttempt) < c.minRefreshInterval {
		return
	}

	c.lastRefreshAttempt = time.Now()
	c.fetchJWKS(true)
}

// fetchJWKS performs the actual JWKS fetch
func (c *JWKSClient) fetchJWKS(preserveOnError bool) (*JWKSCache, error) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// Fetch fresh JWKS
	response, err := c.httpClient.Get(c.jwksURL)
	if err != nil {
		c.cache.lastError = fmt.Errorf("failed to fetch JWKS from %s: %w", c.jwksURL, err)
		c.cache.lastFetchOK = false
		c.cache.fetchedAt = time.Now() // Record that we attempted to fetch
		if preserveOnError && len(c.cache.keysByID) > 0 {
			return c.cache, nil // Return cached keys despite error
		}
		return nil, c.cache.lastError
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		c.cache.lastError = fmt.Errorf("JWKS endpoint returned status: %d", response.StatusCode)
		c.cache.lastFetchOK = false
		c.cache.fetchedAt = time.Now() // Record that we attempted to fetch
		if preserveOnError && len(c.cache.keysByID) > 0 {
			return c.cache, nil // Return cached keys despite error
		}
		return nil, c.cache.lastError
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		c.cache.lastError = fmt.Errorf("failed to read JWKS response: %w", err)
		c.cache.lastFetchOK = false
		c.cache.fetchedAt = time.Now() // Record that we attempted to fetch
		if preserveOnError && len(c.cache.keysByID) > 0 {
			return c.cache, nil // Return cached keys despite error
		}
		return nil, c.cache.lastError
	}

	// Parse Cache-Control header for TTL
	ttlFromHeader := c.parseCacheControlMaxAge(response.Header.Get("Cache-Control"))
	if ttlFromHeader > 0 {
		c.cache.ttl = ttlFromHeader
	}

	// Parse JWKS format
	var jwksResponse JWKSResponse
	if err := json.Unmarshal(responseBody, &jwksResponse); err != nil {
		c.cache.lastError = fmt.Errorf("failed to parse JWKS response: %w", err)
		c.cache.lastFetchOK = false
		if preserveOnError && len(c.cache.keysByID) > 0 {
			return c.cache, nil // Return cached keys despite error
		}
		return nil, c.cache.lastError
	}

	// Convert JWKS to key map, preserving existing keys
	newKeysByID := make(map[string]*rsa.PublicKey)
	if preserveOnError {
		// Copy existing keys
		for kid, key := range c.cache.keysByID {
			newKeysByID[kid] = key
		}
	}

	for _, key := range jwksResponse.Keys {
		if key.KeyType == "RSA" {
			rsaKey, err := c.parseRSAKey(key)
			if err != nil {
				// Log warning but continue with other keys
				continue
			}
			newKeysByID[key.KeyID] = rsaKey
		}
	}

	// Update cache
	now := time.Now()
	c.cache.fetchedAt = now
	c.cache.keysByID = newKeysByID
	c.cache.lastError = nil
	c.cache.lastFetchOK = true

	// Set refresh time to 80% of TTL to start background refresh early
	refreshDelay := time.Duration(float64(c.cache.ttl) * 0.8)
	c.cache.refreshAfter = now.Add(refreshDelay)

	return c.cache, nil
}

// parseCacheControlMaxAge parses max-age from Cache-Control header
func (c *JWKSClient) parseCacheControlMaxAge(cacheControl string) time.Duration {
	if cacheControl == "" {
		return 0
	}

	directives := strings.Split(cacheControl, ",")
	for _, directive := range directives {
		directive = strings.TrimSpace(directive)
		if strings.HasPrefix(directive, "max-age=") {
			maxAgeStr := strings.TrimPrefix(directive, "max-age=")
			if maxAge, err := strconv.Atoi(maxAgeStr); err == nil && maxAge > 0 {
				return time.Duration(maxAge) * time.Second
			}
		}
	}
	return 0
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
