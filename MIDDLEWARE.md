# Middleware Documentation

The simple-query-server supports a flexible middleware system that allows you to process requests before query execution. Middleware can extract values from HTTP headers, validate JWT tokens, and inject parameters into SQL queries.

## Configuration

Middleware is configured using an optional server configuration file (`server.yaml`):

```yaml
middleware:
  # HTTP header middleware - extracts values from HTTP headers and injects as SQL parameters
  - type: "http-header"
    config:
      header: "X-User-ID"      # HTTP header name to extract
      parameter: "user_id"     # SQL parameter name to inject
      required: false          # Whether the header is required (default: false)
  
  # JWT/JWKS verification middleware - verifies JWT tokens and extracts claims
  - type: "bearer-jwks"
    config:
      jwks_url: "http://localhost:3000/.well-known/jwks.json"  # JWKS endpoint URL
      required: false                                          # Whether auth is mandatory
      cache_ttl: "10m"                                         # Cache JWKS keys for 10 minutes (optional, default: 10m)
      enable_health_check: true                                # Include JWKS health in server health checks (optional, default: true)
      claims_mapping:                                          # Map JWT claims to SQL parameters
        sub: "user_id"                                         # Map 'sub' claim to 'user_id' parameter
        role: "user_role"                                      # Map 'role' claim to 'user_role' parameter
        email: "user_email"                                    # Map 'email' claim to 'user_email' parameter
      issuer: "http://localhost:3000"                         # Expected issuer (optional)
      audience: "dev-api"                                      # Expected audience (optional)
  
  # Multiple middleware can be chained
  - type: "http-header"
    config:
      header: "X-Tenant-ID"
      parameter: "tenant_id"
      required: true           # Server returns 400 if header is missing
```

## Middleware Types

### HTTP Header Middleware (`http-header`)

Extracts HTTP header values and makes them available as SQL parameters.

**Configuration:**
- `header`: Name of the HTTP header to extract
- `parameter`: Name of the SQL parameter to inject the header value into
- `required`: Whether the header is required (if true, returns 400 Bad Request when missing)

**Example:**
```bash
# Request with HTTP header middleware parameter
curl -X POST -H "X-User-ID: 123" -H "Content-Type: application/json" \
     -d '{}' http://localhost:8080/query/get_current_user
```

### JWT/JWKS Authentication Middleware (`bearer-jwks`)

Verifies JWT tokens using JWKS endpoints and extracts claims as SQL parameters.

**Configuration:**
- `jwks_url`: URL to fetch JWKS from (e.g., `http://localhost:3000/.well-known/jwks.json`)
- `required`: Whether authentication is mandatory (if true, returns 401 Unauthorized when missing/invalid)
- `cache_ttl`: How long to cache JWKS keys (default: 10 minutes)
- `claims_mapping`: Map JWT claims to SQL parameter names (e.g., `{"sub": "user_id", "role": "user_role"}`)
- `issuer`: Expected JWT issuer for validation (optional)
- `audience`: Expected JWT audience for validation (optional)
- `enable_health_check`: Whether to include JWKS health in server health checks (optional, default: true)

**Authentication Modes:**
- **Optional Authentication**: Set `required: false` to allow requests without tokens
- **Required Authentication**: Set `required: true` to reject unauthenticated requests  

**Example:**
```bash
# Request with JWT authentication (optional)
curl -X POST -H "Authorization: Bearer <jwt_token>" -H "Content-Type: application/json" \
     -d '{}' http://localhost:8080/query/get_user_profile

# Request mixing JWT claims with body parameters  
curl -X POST -H "Authorization: Bearer <jwt_token>" -H "Content-Type: application/json" \
     -d '{"category": "public"}' http://localhost:8080/query/search_user_content
```

## JWKS Health Check

The JWKS middleware supports health checking as part of the server's overall health status.

### Configuration

Add the `enable_health_check` option to your JWKS middleware configuration:

```yaml
middleware:
  - type: "bearer-jwks"
    config:
      jwks_url: "https://your-auth-provider.com/.well-known/jwks.json"
      required: false
      enable_health_check: true  # Enable health check (default: true)
      claims_mapping:
        sub: "user_id"
        role: "user_role"
      issuer: "https://your-auth-provider.com"
      audience: "your-api"
      fallback_ttl: "10m"
```

### Health Check Behavior

**When enabled (default):**
- The server's `/health` endpoint includes middleware status
- Server is healthy only when JWKS middleware has valid, unexpired keys
- JWKS middleware is considered healthy if:
  - Initial JWKS fetch completed successfully
  - Cache is not expired (current time < fetchedAt + ttl)
  - Has valid keys available

**When disabled:**
- JWKS middleware health is ignored in server health checks
- Server health depends only on database and other components
- No middleware section appears in health response

### Health Response Format

**With healthy JWKS middleware:**
```json
{
  "status": "healthy",
  "database": {
    "connected": true
  },
  "middleware": {
    "bearer-jwks(https://your-auth-provider.com/.well-known/jwks.json)": {
      "healthy": true
    }
  }
}
```

**With unhealthy JWKS middleware:**
```json
{
  "status": "unhealthy",
  "database": {
    "connected": true
  },
  "middleware": {
    "bearer-jwks(https://your-auth-provider.com/.well-known/jwks.json)": {
      "healthy": false
    }
  }
}
```

**With health check disabled:**
```json
{
  "status": "healthy",
  "database": {
    "connected": true
  }
}
```

### Use Cases

**Enable health check (default)** when:
- You want load balancers to route traffic away when JWKS is unavailable
- Authentication is critical to your application's functionality
- You need to monitor JWKS endpoint availability

**Disable health check** when:
- JWKS is optional (authentication not required)
- You want the server to remain "healthy" even with JWKS issues
- You prefer to handle JWKS failures gracefully without affecting load balancing

## How Middleware Works

1. Middleware processes requests in the order configured
2. Each middleware can inject additional parameters into the request
3. Parameters are merged with JSON request body parameters
4. Merged parameters are validated against query parameter definitions
5. Query is executed with the combined parameter set

## Chaining Multiple Middleware

Multiple middleware can be configured together:

```bash
# Request with both header and JWT middleware
curl -X POST -H "X-Tenant-ID: acme" -H "Authorization: Bearer <jwt_token>" \
     -H "Content-Type: application/json" -d '{}' \
     http://localhost:8080/query/get_tenant_user_data
```

## JWT/JWKS Authentication Setup

For development and testing with JWT authentication, you can use the included [JWKS Mock API](https://github.com/shogotsuneto/jwks-mock-api):

```bash
# Start JWKS Mock API along with PostgreSQL (provides JWKS endpoint and token generation)
docker compose up -d jwks-mock-api

# Generate a test JWT token
JWT_TOKEN=$(curl -s -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{
    "claims": {
      "sub": "123",
      "role": "admin", 
      "email": "user@example.com"
    },
    "expiresIn": 3600
  }' | jq -r '.token')

# Use the token in requests
curl -X POST -H "Authorization: Bearer $JWT_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{}' http://localhost:8080/query/get_user_profile
```

## Using Middleware with the Server

To use middleware, provide the server configuration file when starting the server:

```bash
# Start the server with middleware support
./server --db-config ./example/database.yaml \
         --queries-config ./example/queries.yaml \
         --server-config ./example/server.yaml
```

## Caching and Performance

The JWT/JWKS middleware includes intelligent caching:

- **JWKS Key Caching**: Public keys are cached with configurable TTL (default: 10 minutes)
- **Thread-Safe**: Cache operations are protected with mutexes
- **Automatic Invalidation**: Keys are refreshed when the cache expires
- **Multiple Key Formats**: Supports both X.509 certificates and modulus/exponent formats

This reduces external JWKS requests and improves performance for high-traffic applications.