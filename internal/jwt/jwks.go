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
	jwksURL            string
	cache              *JWKSCache
	cacheMutex         sync.RWMutex
	httpClient         *http.Client
	fallbackTTL        time.Duration
	lastRefetchAttempt time.Time
	refetchMutex       sync.Mutex
	refetchMinInterval time.Duration // minimum interval between refetch attempts
}

// NewJWKSClient creates a new JWKS client with configurable fallback TTL
func NewJWKSClient(jwksURL string, fallbackTTL time.Duration) *JWKSClient {
	return &JWKSClient{
		jwksURL: jwksURL,
		cache: &JWKSCache{
			keysByID: make(map[string]*rsa.PublicKey),
			ttl:      fallbackTTL,
		},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		fallbackTTL:        fallbackTTL,
		refetchMinInterval: 30 * time.Second, // prevent excessive refetch attempts
	}
}

// GetPublicKey retrieves the public key for the given key ID
func (c *JWKSClient) GetPublicKey(kid string) (*rsa.PublicKey, error) {
	cache, err := c.fetchJWKSWithCache(kid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	rsaPublicKey, exists := cache.keysByID[kid]
	if !exists {
		return nil, fmt.Errorf("key not found for kid: %s", kid)
	}

	return rsaPublicKey, nil
}

// fetchJWKSWithCache fetches JWKS with caching logic and handles refetching for unknown keys
func (c *JWKSClient) fetchJWKSWithCache(kid string) (*JWKSCache, error) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	// Check if cache is still valid
	if c.isCacheValid() {
		// If looking for a specific key and it's not in cache, try refetch (with rate limiting)
		if kid != "" {
			if _, exists := c.cache.keysByID[kid]; !exists {
				if c.shouldAttemptRefetchLocked() {
					// Invalidate cache to force refetch
					c.cache.fetchedAt = time.Time{}
				} else {
					// Rate limited, return current cache
					return c.cache, nil
				}
			} else {
				// Key found in valid cache
				return c.cache, nil
			}
		} else {
			// No specific key requested, return valid cache
			return c.cache, nil
		}
	}

	// Fetch fresh JWKS
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

	// Update cache
	c.cache.fetchedAt = time.Now()
	c.cache.keysByID = keysByID
	c.cache.ttl = cacheTTL

	return c.cache, nil
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

// isCacheValid checks if the current cache is still valid
func (c *JWKSClient) isCacheValid() bool {
	if c.cache.keysByID == nil || len(c.cache.keysByID) == 0 {
		return false
	}

	valid := time.Since(c.cache.fetchedAt) < c.cache.ttl
	return valid
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

// shouldAttemptRefetchLocked atomically checks if we should attempt to refetch JWKS for unknown key
// and updates lastRefetchAttempt if true to prevent multiple simultaneous refetches
// NOTE: This method expects the caller to hold the refetchMutex
func (c *JWKSClient) shouldAttemptRefetchLocked() bool {
	c.refetchMutex.Lock()
	defer c.refetchMutex.Unlock()

	if time.Since(c.lastRefetchAttempt) >= c.refetchMinInterval {
		c.lastRefetchAttempt = time.Now()
		return true
	}
	return false
}
