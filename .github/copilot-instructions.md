# Simple Query Server - GitHub Copilot Instructions

**CRITICAL: Always follow these instructions exactly. Only search for additional information or run exploratory bash commands if the specific information you need is incomplete or found to be incorrect in these instructions. These instructions contain validated, working commands and procedures.**

## Overview

The `simple-query-server` is a lightweight Go HTTP server that exposes database queries defined in YAML configuration files as REST API endpoints. It currently uses mock responses for demonstration purposes but is designed to support PostgreSQL, MySQL, and SQLite databases.

## Code Change Guidelines

**Documentation Updates:**
- Always update documentation (including this copilot-instructions.md file) when making changes that affect:
  - Build processes, commands, or timing
  - API endpoints or behavior
  - Configuration file formats
  - Development workflow or procedures
  - New features or significant changes to existing functionality

**Backward Compatibility:**
- Backward compatibility is usually not a concern unless explicitly told to maintain compatibility
- Feel free to make breaking changes to improve the codebase when beneficial
- Only preserve backward compatibility when specifically requested or when working with public APIs that external users depend on

## Working Effectively

### Prerequisites and Setup
- Go 1.18 or later is required (Go 1.24.6 confirmed working)
- No additional dependencies beyond what's in go.mod

### Makefile Commands
A comprehensive Makefile is available with convenient shortcuts for all development tasks. Run `make help` to see all available commands organized by category:
- **Development:** deps, clean-deps, build, clean, vet, fmt, fmt-check, test, all
- **Running:** run, run-test, run-help  
- **API Testing:** health, queries, api-test

Use make commands when possible as they provide consistent, validated workflows.

### Essential Commands

**Download dependencies:**
```bash
make deps
# OR: go mod download
```
- Takes <1 second - use default timeout

**Clean dependencies:**
```bash
make clean-deps
# OR: go mod tidy
```  
- Takes <1 second - use default timeout

**Clear build cache (for testing clean builds):**
```bash
make clean-cache
# OR: go clean -cache -modcache
```
- Takes <1 second - use default timeout
- Forces next build to download all dependencies

**Build the binary:**
```bash
make build
# OR: go build -o server ./cmd/server
```
- First build (clean): Takes ~11 seconds. NEVER CANCEL. Set timeout to 60+ seconds.
- Subsequent builds (cached): Takes <1 second

**Run code validation:**
```bash
make vet
# OR: go vet ./...
```
- Takes ~2 seconds - use default timeout

**Check code formatting:**
```bash
make fmt-check
# OR: gofmt -l .
```
- Takes <1 second - use default timeout
- Lists files that need formatting (some files in this repo are not perfectly formatted but this is non-critical)

**Fix formatting:**
```bash
make fmt
# OR: gofmt -w .
```
- Takes <1 second - use default timeout

**Run tests (no tests currently exist):**
```bash
make test
# OR: go test ./...
```
- Takes ~2 seconds - use default timeout
- Currently returns "no test files" for all packages

### Running the Application

**Start the server with example configuration:**
```bash
make run
# OR: ./server --db-config ./example/database.yaml --queries-config ./example/queries.yaml --port 8080
```
- Server starts in ~3 seconds
- Requires both --db-config and --queries-config flags
- Default port is 8080 if not specified

**Start with test configuration:**
```bash
make run-test
# OR: ./server --db-config ./testdata/database.yaml --queries-config ./testdata/queries.yaml --port 8081
```

**View help:**
```bash
make run-help
# OR: ./server --help
```

## API Validation

After starting the server, ALWAYS test functionality with these validation scenarios:

**Run comprehensive API tests:**
```bash
make api-test
```
This runs all 7 validation scenarios below automatically. For individual tests:

### Health Check
```bash
make health
# OR: curl http://localhost:8080/health
```
Expected response: `{"status":"healthy"}`

### List Available Queries
```bash
make queries
# OR: curl http://localhost:8080/queries
```
Expected response: JSON object with all configured queries and their parameters

### Execute Query with Parameters
```bash
curl -X POST -H "Content-Type: application/json" -d '{"id": 123}' http://localhost:8080/query/get_user_by_id
```
Expected response: `{"rows":[{"email":"user123@example.com","id":123,"name":"User 123"}]}`

### Execute Query without Parameters
```bash
curl -X POST -H "Content-Type: application/json" -d '{}' http://localhost:8080/query/get_all_active_users
```
Expected response: JSON with mock user data

### Test Search Query
```bash
curl -X POST -H "Content-Type: application/json" -d '{"name": "%Alice%"}' http://localhost:8080/query/search_users
```
Expected response: JSON with mock Alice user data

### Test Error Handling
```bash
curl -X POST -H "Content-Type: application/json" -d '{"invalid": "param"}' http://localhost:8080/query/get_user_by_id
```
Expected response: `{"error":"required parameter 'id' is missing"}`

```bash
curl -X POST -H "Content-Type: application/json" -d '{}' http://localhost:8080/query/nonexistent_query  
```
Expected response: `{"error":"Query 'nonexistent_query' not found"}`

### Test Invalid Configuration
```bash
./server --db-config ./nonexistent.yaml --queries-config ./example/queries.yaml
```
Expected: Exit with error about missing database config file

```bash
./server
```
Expected: Exit with usage message requiring both config flags

## Configuration Files

The server requires two YAML configuration files:

### Database Configuration (database.yaml)
- Defines database connection settings (type, DSN, credentials)
- Example locations: `./example/database.yaml`, `./testdata/database.yaml`
- Currently supports postgres, mysql, sqlite (mock responses only)

### Queries Configuration (queries.yaml)  
- Defines available queries with SQL and parameter definitions
- Example locations: `./example/queries.yaml`, `./testdata/queries.yaml`
- Each query has a name, SQL statement, and parameter list with types

## Project Structure

```
simple-query-server/
├── cmd/server/main.go           # Main entry point - CLI flag handling and server startup
├── internal/config/loader.go    # YAML configuration loading and validation
├── internal/query/executor.go   # Query execution engine (currently mock responses)
├── internal/server/http.go      # HTTP server and REST API routing
├── example/                     # Example configuration files for demo
├── testdata/                    # Test configuration files
└── go.mod                       # Go module definition
```

## Key Implementation Details

- **Mock Responses**: All database queries currently return mock data based on SQL pattern matching
- **Parameter Validation**: Automatic validation of query parameters with type checking
- **Error Handling**: Comprehensive error responses for missing configs, invalid queries, etc.
- **No Tests**: Currently no automated tests exist - manual API validation is required

## Development Workflow

1. **Build and validate changes:**
   ```bash
   make all
   # OR: go build -o server ./cmd/server && go vet ./...
   ```

2. **Test your changes:**
   - Start server: `make run`
   - Run API validation: `make api-test`
   - Verify responses match expected output

3. **Always test error scenarios:**
   - Missing configuration files
   - Invalid query names
   - Missing required parameters
   - Invalid JSON in request body

4. **Before committing:**
   ```bash
   make clean-deps && make fmt
   # OR: go mod tidy && gofmt -w .
   ```

## Common Issues

- **Server won't start**: Check that both --db-config and --queries-config flags are provided
- **Query not found**: Verify query name matches exactly what's defined in queries.yaml
- **Parameter validation error**: Check parameter names and types match query definition
- **Build failures**: Ensure Go 1.18+ is installed and go.mod is clean

## Important Notes

- NEVER CANCEL builds - first build takes ~11 seconds, subsequent builds <1 second
- Always test with actual API calls after making changes - simply starting/stopping the server is insufficient
- ALWAYS run through complete end-to-end validation scenarios after making changes
- The server uses mock responses - no actual database connection is made
- Configuration files must be valid YAML with proper structure
- Parameter binding uses `:parameter_name` syntax in SQL queries
- Clean builds require downloading dependencies: `go clean -cache -modcache` before building