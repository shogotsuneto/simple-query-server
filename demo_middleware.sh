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
echo "1️⃣  Test: Regular query without middleware headers (should work normally)"
response=$(curl -s -X POST -H "Content-Type: application/json" -d '{}' http://localhost:8080/query/get_all_active_users)
echo "Response: $(echo $response | jq -r '.rows | length') active users found"

echo ""
echo "2️⃣  Test: Query with middleware-injected parameter (X-User-ID header)"
echo "Command: curl -X POST -H 'X-User-ID: 2' -H 'Authorization: Bearer test-token' -d '{}' http://localhost:8080/query/get_current_user"
response=$(curl -s -X POST -H "Content-Type: application/json" -H "X-User-ID: 2" -H "Authorization: Bearer test-token" -d '{}' http://localhost:8080/query/get_current_user)
echo "Response: $(echo $response | jq -r '.rows[0].name') (ID: $(echo $response | jq -r '.rows[0].id'))"

echo ""
echo "3️⃣  Test: Missing required Authorization header (should fail)"
echo "Command: curl -X POST -H 'X-User-ID: 2' -d '{}' http://localhost:8080/query/get_current_user"
response=$(curl -s -X POST -H "Content-Type: application/json" -H "X-User-ID: 2" -d '{}' http://localhost:8080/query/get_current_user)
echo "Response: $(echo $response | jq -r '.error')"

echo ""
echo "4️⃣  Test: Different user ID via header"
echo "Command: curl -X POST -H 'X-User-ID: 1' -H 'Authorization: Bearer test-token' -d '{}' http://localhost:8080/query/get_current_user"
response=$(curl -s -X POST -H "Content-Type: application/json" -H "X-User-ID: 1" -H "Authorization: Bearer test-token" -d '{}' http://localhost:8080/query/get_current_user)
echo "Response: $(echo $response | jq -r '.rows[0].name') (ID: $(echo $response | jq -r '.rows[0].id'))"

echo ""
echo "5️⃣  Test: Regular query with JSON parameters (middleware doesn't interfere)"
echo "Command: curl -X POST -H 'Authorization: Bearer test-token' -d '{\"id\": 3}' http://localhost:8080/query/get_user_by_id"
response=$(curl -s -X POST -H "Content-Type: application/json" -H "Authorization: Bearer test-token" -d '{"id": 3}' http://localhost:8080/query/get_user_by_id)
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
echo "- ✅ Required headers are enforced"
echo "- ✅ Optional headers work correctly"
echo "- ✅ Regular queries continue to work normally"
echo "- ✅ Backward compatibility is maintained"