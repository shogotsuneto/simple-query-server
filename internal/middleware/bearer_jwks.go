package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shogotsuneto/simple-query-server/internal/jwt"
)

// BearerJWKSConfig represents the configuration for bearer JWKS middleware
type BearerJWKSConfig struct {
	JWKSURL       string            `yaml:"jwks_url"`       // URL to fetch JWKS from
	Required      bool              `yaml:"required"`       // Whether authentication is mandatory
	ClaimsMapping map[string]string `yaml:"claims_mapping"` // Map JWT claims to SQL parameters
	Issuer        string            `yaml:"issuer"`         // Expected issuer for validation (optional)
	Audience      string            `yaml:"audience"`       // Expected audience for validation (optional)
	FallbackTTL   string            `yaml:"fallback_ttl"`   // Fallback TTL for JWKS when no Cache-Control header (optional, default: 10m)
}

// BearerJWKSMiddleware verifies JWT tokens using JWKS and injects claims as SQL parameters
type BearerJWKSMiddleware struct {
	config     BearerJWKSConfig
	jwksClient *jwt.JWKSClient
}

// NewBearerJWKSMiddleware creates a new bearer JWKS middleware
func NewBearerJWKSMiddleware(config BearerJWKSConfig) *BearerJWKSMiddleware {
	// Parse fallback TTL, default to 10 minutes
	fallbackTTL := 10 * time.Minute
	if config.FallbackTTL != "" {
		if parsedTTL, err := time.ParseDuration(config.FallbackTTL); err == nil {
			fallbackTTL = parsedTTL
		}
	}

	return &BearerJWKSMiddleware{
		config:     config,
		jwksClient: jwt.NewJWKSClient(config.JWKSURL, fallbackTTL),
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
		claims, err := m.jwksClient.ValidateToken(tokenString, m.config.Issuer, m.config.Audience)
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

// Name returns the name of this middleware
func (m *BearerJWKSMiddleware) Name() string {
	return fmt.Sprintf("bearer-jwks(%s)", m.config.JWKSURL)
}
