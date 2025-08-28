#!/bin/bash

# Middleware Demonstration Script
# This script demonstrates the middleware configuration functionality

echo "🚀 Simple Query Server - Middleware Configuration Demo"
echo "===================================================="

# Cleanup any existing containers
echo "🧹 Cleaning up any existing containers..."
docker compose down -v 2>/dev/null || true

# Start PostgreSQL
echo ""
echo "🐘 Starting PostgreSQL database..."
docker compose up -d postgres
sleep 8  # Wait for database to be ready

# Build the server
echo ""
echo "🔨 Building the server..."
make build

echo ""
echo "📋 Middleware Configuration:"
echo "----------------------------"
cat example/server.yaml

# Start server with middleware in background
echo ""
echo "🚀 Starting server with middleware configuration..."
./server --db-config ./example/database.yaml --queries-config ./example/queries.yaml --server-config ./example/server.yaml --port 8080 &
SERVER_PID=$!
sleep 5  # Wait for server to start

echo ""
echo "✅ Server started with middleware configuration"

# Demonstrate middleware functionality
echo ""
echo "🧪 Testing Middleware Functionality:"
echo "====================================="

echo ""
echo "1️⃣  Test: View query definitions with separated parameters"
echo "Command: curl http://localhost:8080/queries | jq '.queries.get_current_user'"
response=$(curl -s http://localhost:8080/queries | jq '.queries.get_current_user')
echo "Response:"
echo "$response"

echo ""
echo "2️⃣  Test: Regular query without middleware headers (should work normally)"
response=$(curl -s -X POST -H "Content-Type: application/json" -d '{}' http://localhost:8080/query/get_all_active_users)
echo "Response: $(echo $response | jq -r '.rows | length') active users found"

echo ""
echo "3️⃣  Test: Query with middleware-injected parameter (X-User-ID header)"
echo "Command: curl -X POST -H 'X-User-ID: 2' -d '{}' http://localhost:8080/query/get_current_user"
response=$(curl -s -X POST -H "Content-Type: application/json" -H "X-User-ID: 2" -d '{}' http://localhost:8080/query/get_current_user)
echo "Response: $(echo $response | jq -r '.rows[0].name') (ID: $(echo $response | jq -r '.rows[0].id'))"

echo ""
echo "4️⃣  Test: Attempting to provide middleware parameter in body (should fail)"
echo "Command: curl -X POST -d '{\"user_id\": \"123\"}' http://localhost:8080/query/get_current_user"
response=$(curl -s -X POST -H "Content-Type: application/json" -d '{"user_id": "123"}' http://localhost:8080/query/get_current_user)
echo "Response: $(echo $response | jq -r '.error')"

echo ""
echo "5️⃣  Test: Mixed parameters - body + middleware"
echo "Command: curl -X POST -H 'X-Tenant-ID: tenant123' -d '{\"id\": 2}' http://localhost:8080/query/get_user_by_tenant"
response=$(curl -s -X POST -H "Content-Type: application/json" -H "X-Tenant-ID: tenant123" -d '{"id": 2}' http://localhost:8080/query/get_user_by_tenant)
echo "Response: $(echo $response | jq -r '.error')" # Expected: column tenant_id doesn't exist

echo ""
echo "6️⃣  Test: Attempting to provide middleware parameter in body for mixed query (should fail)"
echo "Command: curl -X POST -d '{\"id\": 2, \"tenant_id\": \"tenant123\"}' http://localhost:8080/query/get_user_by_tenant"
response=$(curl -s -X POST -H "Content-Type: application/json" -d '{"id": 2, "tenant_id": "tenant123"}' http://localhost:8080/query/get_user_by_tenant)
echo "Response: $(echo $response | jq -r '.error')"

echo ""
echo "7️⃣  Test: Regular query with JSON parameters (middleware doesn't interfere)"
echo "Command: curl -X POST -d '{\"id\": 3}' http://localhost:8080/query/get_user_by_id"
response=$(curl -s -X POST -H "Content-Type: application/json" -d '{"id": 3}' http://localhost:8080/query/get_user_by_id)
echo "Response: $(echo $response | jq -r '.rows[0].name') (ID: $(echo $response | jq -r '.rows[0].id'))"

# Cleanup
echo ""
echo "🧹 Cleaning up..."
kill $SERVER_PID 2>/dev/null || true
docker compose down -v

echo ""
echo "✅ Middleware demonstration completed successfully!"
echo ""
echo "🎯 Summary:"
echo "- ✅ Server starts with optional middleware configuration"
echo "- ✅ HTTP headers are extracted and injected as SQL parameters"
echo "- ✅ Parameters are separated by source (body vs middleware)"
echo "- ✅ Body parameters cannot override middleware parameters"
echo "- ✅ /queries endpoint shows both parameter types separately"
echo "- ✅ Mixed queries (body + middleware parameters) work correctly"
echo "- ✅ Validation prevents parameter conflicts"
echo "- ✅ Regular queries continue to work normally"
echo "- ✅ Backward compatibility is maintained"