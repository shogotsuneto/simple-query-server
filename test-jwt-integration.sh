#!/bin/bash
# Manual integration test for JWT middleware with JWKS mock API

set -e

# Start JWKS mock API in the background
echo "Starting JWKS mock API..."
./test-tools/jwks-mock-api &
JWKS_PID=$!

# Wait for JWKS mock API to start
sleep 3

# Function to cleanup
cleanup() {
    echo "Cleaning up..."
    if [ ! -z "$JWKS_PID" ]; then
        kill $JWKS_PID 2>/dev/null || true
    fi
    if [ ! -z "$SERVER_PID" ]; then
        kill $SERVER_PID 2>/dev/null || true
    fi
    if [ ! -z "$POSTGRES_CONTAINER" ]; then
        docker stop $POSTGRES_CONTAINER 2>/dev/null || true
    fi
}

trap cleanup EXIT

# Check if JWKS mock API is running
echo "Checking JWKS mock API health..."
curl -f http://localhost:3000/health || {
    echo "JWKS mock API failed to start"
    exit 1
}

echo "JWKS mock API is running!"

# Get JWKS to verify it's working
echo "Fetching JWKS..."
curl -s http://localhost:3000/.well-known/jwks.json | jq .

# Generate a test JWT token
echo "Generating test JWT token..."
JWT_RESPONSE=$(curl -s -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{
    "claims": {
      "sub": "2",
      "role": "admin", 
      "email": "admin@example.com"
    },
    "expiresIn": 3600
  }')

echo "JWT Response: $JWT_RESPONSE"

# Extract the token
JWT_TOKEN=$(echo "$JWT_RESPONSE" | jq -r '.access_token')

if [ "$JWT_TOKEN" = "null" ] || [ -z "$JWT_TOKEN" ]; then
    echo "Failed to generate JWT token"
    exit 1
fi

echo "Generated JWT token: ${JWT_TOKEN:0:50}..."

# Start PostgreSQL database
echo "Starting PostgreSQL database..."
docker compose up -d postgres
POSTGRES_CONTAINER=$(docker compose ps -q postgres)

# Wait for database to be ready
sleep 10

# Start our server with JWT configuration
echo "Starting simple-query-server with JWT middleware..."
./server --db-config ./integration/config/database.yaml \
         --queries-config ./integration/config/queries-with-jwt.yaml \
         --server-config ./integration/config/server-with-jwt.yaml \
         --port 8081 &
SERVER_PID=$!

# Wait for server to start
sleep 5

# Test 1: Request without authentication (should work since required=false)
echo "Test 1: Request without authentication..."
curl -s -X POST -H "Content-Type: application/json" \
     -d '{}' http://localhost:8081/query/get_all_active_users | jq .

# Test 2: Request with JWT authentication
echo "Test 2: Request with JWT authentication..."
AUTH_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
     -H "Authorization: Bearer $JWT_TOKEN" \
     -d '{}' http://localhost:8081/query/get_user_by_jwt_sub)

echo "Auth response: $AUTH_RESPONSE"

# Test 3: Request with JWT authentication and claims verification
echo "Test 3: Request with JWT claims verification..."
CLAIMS_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
     -H "Authorization: Bearer $JWT_TOKEN" \
     -d '{}' http://localhost:8081/query/get_user_profile_with_claims)

echo "Claims response: $CLAIMS_RESPONSE"

# Test 4: Request mixing JWT auth with body parameters
echo "Test 4: Request mixing JWT auth with body parameters..."
MIXED_RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
     -H "Authorization: Bearer $JWT_TOKEN" \
     -d '{"search_term": "%Bob%"}' http://localhost:8081/query/search_with_auth)

echo "Mixed parameters response: $MIXED_RESPONSE"

echo "All tests completed successfully!"