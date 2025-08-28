package middleware

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// BearerJWKSConfig represents the configuration for bearer JWKS middleware
type BearerJWKSConfig struct {
	JWKSURL       string            `yaml:"jwks_url"`       // URL to fetch JWKS from
	Required      bool              `yaml:"required"`       // Whether authentication is mandatory
	ClaimsMapping map[string]string `yaml:"claims_mapping"` // Map JWT claims to SQL parameters
	Issuer        string            `yaml:"issuer"`         // Expected issuer for validation (optional)
	Audience      string            `yaml:"audience"`       // Expected audience for validation (optional)
}

// BearerJWKSMiddleware verifies JWT tokens using JWKS and injects claims as SQL parameters
type BearerJWKSMiddleware struct {
	config   BearerJWKSConfig
	keyCache map[string]*rsa.PublicKey
	cacheMu  sync.RWMutex
	client   *http.Client
}

// NewBearerJWKSMiddleware creates a new bearer JWKS middleware
func NewBearerJWKSMiddleware(config BearerJWKSConfig) *BearerJWKSMiddleware {
	return &BearerJWKSMiddleware{
		config:   config,
		keyCache: make(map[string]*rsa.PublicKey),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Wrap wraps an http.HandlerFunc with this middleware
func (m *BearerJWKSMiddleware) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract Authorization header
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			if m.config.Required {
				http.Error(w, "Authorization header is required", http.StatusUnauthorized)
				return
			}
			// If not required and missing, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Check for Bearer token format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			if m.config.Required {
				http.Error(w, "Authorization header must be a Bearer token", http.StatusUnauthorized)
				return
			}
			// If not required and malformed, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from "Bearer <token>"
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		if tokenString == "" {
			if m.config.Required {
				http.Error(w, "Bearer token is empty", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		// Parse and validate the JWT token
		claims, err := m.validateToken(tokenString)
		if err != nil {
			if m.config.Required {
				http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
				return
			}
			// If not required and invalid, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Get existing middleware parameters from context
		params := GetMiddlewareParams(r)

		// Map JWT claims to SQL parameters according to configuration
		for jwtClaim, sqlParam := range m.config.ClaimsMapping {
			if claimValue, exists := claims[jwtClaim]; exists {
				params[sqlParam] = claimValue
			}
		}

		// Set updated parameters in context and continue
		r = SetMiddlewareParams(r, params)
		next.ServeHTTP(w, r)
	}
}

// validateToken parses and validates a JWT token against JWKS
func (m *BearerJWKSMiddleware) validateToken(tokenString string) (jwt.MapClaims, error) {
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
		publicKey, err := m.getPublicKey(kid)
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

	// Validate issuer if configured
	if m.config.Issuer != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != m.config.Issuer {
			return nil, fmt.Errorf("invalid issuer: expected %s, got %v", m.config.Issuer, claims["iss"])
		}
	}

	// Validate audience if configured
	if m.config.Audience != "" {
		if aud, ok := claims["aud"].(string); !ok || aud != m.config.Audience {
			return nil, fmt.Errorf("invalid audience: expected %s, got %v", m.config.Audience, claims["aud"])
		}
	}

	return claims, nil
}

// getPublicKey retrieves the public key for the given key ID from JWKS
func (m *BearerJWKSMiddleware) getPublicKey(kid string) (*rsa.PublicKey, error) {
	// Check cache first
	m.cacheMu.RLock()
	if key, exists := m.keyCache[kid]; exists {
		m.cacheMu.RUnlock()
		return key, nil
	}
	m.cacheMu.RUnlock()

	// Fetch JWKS using the jwx library
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keySet, err := jwk.Fetch(ctx, m.config.JWKSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	// Find the key with matching kid
	key, found := keySet.LookupKeyID(kid)
	if !found {
		return nil, fmt.Errorf("key with id %s not found in JWKS", kid)
	}

	// Convert to RSA public key
	var rsaKey rsa.PublicKey
	if err := key.Raw(&rsaKey); err != nil {
		return nil, fmt.Errorf("failed to convert key to RSA public key: %w", err)
	}

	// Cache the key
	m.cacheMu.Lock()
	m.keyCache[kid] = &rsaKey
	m.cacheMu.Unlock()

	return &rsaKey, nil
}

// Name returns the name of this middleware
func (m *BearerJWKSMiddleware) Name() string {
	return fmt.Sprintf("bearer-jwks(%s)", m.config.JWKSURL)
}
