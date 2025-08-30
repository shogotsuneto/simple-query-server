# JWKS Middleware Health Check Configuration

The JWKS middleware now supports health checking as part of the server's overall health status.

## Configuration

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

## Health Check Behavior

### When enabled (default):
- The server's `/health` endpoint will include middleware status
- Server is healthy only when JWKS middleware has valid, unexpired keys
- JWKS middleware is considered healthy if:
  - Initial JWKS fetch completed successfully
  - Cache is not expired (current time < fetchedAt + ttl)
  - No recent fetch failures (failureCount == 0)

### When disabled:
- JWKS middleware health is ignored in server health checks
- Server health depends only on database and other components
- No middleware section appears in health response

## Health Response Format

### With healthy JWKS middleware:
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

### With unhealthy JWKS middleware:
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

### With health check disabled:
```json
{
  "status": "healthy",
  "database": {
    "connected": true
  }
}
```

## Use Cases

**Enable health check (default)** when:
- You want load balancers to route traffic away when JWKS is unavailable
- Authentication is critical to your application's functionality
- You need to monitor JWKS endpoint availability

**Disable health check** when:
- JWKS is optional (authentication not required)
- You want the server to remain "healthy" even with JWKS issues
- You prefer to handle JWKS failures gracefully without affecting load balancing