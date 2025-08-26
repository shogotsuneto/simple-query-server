# Simple Query Server Makefile
# This Makefile provides convenient commands for development and testing

.PHONY: help deps build clean vet fmt fmt-check test run run-test run-help api-test health queries clean-cache all

# Default target
help:
	@echo "Simple Query Server - Available Commands:"
	@echo ""
	@echo "Development:"
	@echo "  deps       - Download Go module dependencies"
	@echo "  clean-deps - Clean Go module cache and dependencies"
	@echo "  build      - Build the server binary"
	@echo "  clean      - Clean build artifacts"
	@echo "  vet        - Run Go code validation"
	@echo "  fmt        - Format Go code (fixes formatting)"
	@echo "  fmt-check  - Check Go code formatting (reports issues only)"
	@echo "  test       - Run tests"
	@echo "  all        - Run deps, vet, fmt-check, test, and build"
	@echo ""
	@echo "Running:"
	@echo "  run        - Start server with example configuration (port 8080)"
	@echo "  run-test   - Start server with test configuration (port 8081)"
	@echo "  run-help   - Show server help"
	@echo ""
	@echo "API Testing (requires server to be running):"
	@echo "  health     - Test health endpoint"
	@echo "  queries    - List available queries"
	@echo "  api-test   - Run comprehensive API tests"

# Dependency management
deps:
	go mod download

clean-deps:
	go mod tidy

clean-cache:
	go clean -cache -modcache

# Build
build:
	go build -o server ./cmd/server

clean:
	rm -f server

# Code quality
vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@files=$$(gofmt -l .); if [ -n "$$files" ]; then echo "Files need formatting:"; echo "$$files"; exit 1; fi

test:
	go test ./...

# Comprehensive build and validation
all: deps vet fmt-check test build

# Running the server
run: build
	./server --db-config ./example/database.yaml --queries-config ./example/queries.yaml --port 8080

run-test: build
	./server --db-config ./testdata/database.yaml --queries-config ./testdata/queries.yaml --port 8081

run-help: build
	./server --help

# API testing (requires server to be running on port 8080)
health:
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8080/health | python3 -m json.tool || curl -s http://localhost:8080/health

queries:
	@echo "Listing available queries..."
	@curl -s http://localhost:8080/queries | python3 -m json.tool || curl -s http://localhost:8080/queries

api-test:
	@echo "Running comprehensive API tests..."
	@echo "1. Health check:"
	@curl -s http://localhost:8080/health | python3 -m json.tool || curl -s http://localhost:8080/health
	@echo ""
	@echo "2. List queries:"
	@curl -s http://localhost:8080/queries | python3 -m json.tool || curl -s http://localhost:8080/queries
	@echo ""
	@echo "3. Get user by ID:"
	@curl -s -X POST -H "Content-Type: application/json" -d '{"id": 123}' http://localhost:8080/query/get_user_by_id | python3 -m json.tool || curl -s -X POST -H "Content-Type: application/json" -d '{"id": 123}' http://localhost:8080/query/get_user_by_id
	@echo ""
	@echo "4. Get all active users:"
	@curl -s -X POST -H "Content-Type: application/json" -d '{}' http://localhost:8080/query/get_all_active_users | python3 -m json.tool || curl -s -X POST -H "Content-Type: application/json" -d '{}' http://localhost:8080/query/get_all_active_users
	@echo ""
	@echo "5. Search users:"
	@curl -s -X POST -H "Content-Type: application/json" -d '{"name": "%Alice%"}' http://localhost:8080/query/search_users | python3 -m json.tool || curl -s -X POST -H "Content-Type: application/json" -d '{"name": "%Alice%"}' http://localhost:8080/query/search_users
	@echo ""
	@echo "6. Test error handling - missing parameter:"
	@curl -s -X POST -H "Content-Type: application/json" -d '{"invalid": "param"}' http://localhost:8080/query/get_user_by_id | python3 -m json.tool || curl -s -X POST -H "Content-Type: application/json" -d '{"invalid": "param"}' http://localhost:8080/query/get_user_by_id
	@echo ""
	@echo "7. Test error handling - nonexistent query:"
	@curl -s -X POST -H "Content-Type: application/json" -d '{}' http://localhost:8080/query/nonexistent_query | python3 -m json.tool || curl -s -X POST -H "Content-Type: application/json" -d '{}' http://localhost:8080/query/nonexistent_query