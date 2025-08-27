# Simple Query Server - GitHub Copilot Instructions

**CRITICAL: Always follow these instructions exactly. Only search for additional information or run exploratory bash commands if the specific information you need is incomplete or found to be incorrect in these instructions. These instructions contain validated, working commands and procedures.**

## Overview

The `simple-query-server` is a lightweight Go HTTP server that exposes database queries defined in YAML configuration files as REST API endpoints. It supports PostgreSQL databases with full query execution and parameter binding. **Database connection is required** - the server will fail to start if PostgreSQL is unavailable.

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
- Docker and Docker Compose for PostgreSQL database
- Dependencies: gopkg.in/yaml.v3 and github.com/lib/pq (PostgreSQL driver)

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
```

- Takes <1 second - use default timeout

**Clean dependencies:**

```bash
make clean-deps
```

- Takes <1 second - use default timeout

**Clear build cache (for testing clean builds):**

```bash
make clean-cache
```

- Takes <1 second - use default timeout
- Forces next build to download all dependencies

**Build the binary:**

```bash
make build
```

- First build (clean): Takes ~11 seconds. NEVER CANCEL. Set timeout to 60+ seconds.
- Subsequent builds (cached): Takes <1 second

**Run code validation:**

```bash
make vet
```

- Takes ~2 seconds - use default timeout

**Check code formatting:**

```bash
make fmt-check
```

- Takes <1 second - use default timeout
- Lists files that need formatting (some files in this repo are not perfectly formatted but this is non-critical)

**Fix formatting:**

```bash
make fmt
```

- Takes <1 second - use default timeout

**Run unit tests:**

```bash
make test
```

- Takes ~2 seconds - use default timeout
- Currently includes unit tests for SQL parameter conversion logic

**Run integration tests:**

```bash
make integration-test
```

- Takes ~30 seconds - set timeout to 60+ seconds
- Starts isolated PostgreSQL database in Docker
- Builds and runs server with test configuration
- Executes comprehensive HTTP API tests with real database
- Automatically cleans up test environment

- Takes ~2 seconds - use default timeout
- Currently returns "no test files" for all packages

### Running the Application

**Start PostgreSQL database first:**

```bash
docker compose up -d postgres
```

- Takes ~30 seconds first time (downloads PostgreSQL image)
- Subsequently takes ~5 seconds
- Database is initialized with schema and sample data

**Start the server with example configuration:**

```bash
make run
# OR: ./server --db-config ./example/database.yaml --queries-config ./example/queries.yaml --port 8080
```

- Server starts in ~3 seconds
- **Requires PostgreSQL database to be running** - server will exit with error if database is unavailable
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
curl http://localhost:8080/health
```

Expected response: `{"status":"healthy"}`

### List Available Queries

```bash
curl http://localhost:8080/queries
```

Expected response: JSON object with all configured queries and their parameters

### Execute Query with Parameters

```bash
curl -X POST -H "Content-Type: application/json" -d '{"id": 2}' http://localhost:8080/query/get_user_by_id
```

Expected response: `{"rows":[{"email":"bob.johnson@example.com","id":2,"name":"Bob Johnson"}]}`

### Execute Query without Parameters

```bash
curl -X POST -H "Content-Type: application/json" -d '{}' http://localhost:8080/query/get_all_active_users
```

Expected response: JSON with PostgreSQL user data (19 active users from sample database)

### Test Search Query

```bash
curl -X POST -H "Content-Type: application/json" -d '{"name": "%Alice%"}' http://localhost:8080/query/search_users
```

Expected response: JSON with Alice users from PostgreSQL database (Alice Smith, Alice Johnson, Alice Brown)

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

- Defines database connection settings (type, DSN)
- Example locations: `./example/database.yaml`, `./testdata/database.yaml`
- Currently supports PostgreSQL (full implementation) - database connection is required

### Queries Configuration (queries.yaml)

- Defines available queries with SQL and parameter definitions
- Example locations: `./example/queries.yaml`, `./testdata/queries.yaml`
- Each query has a name, SQL statement, and parameter list with types
- Uses `:parameter_name` syntax for parameter binding

## Project Structure

```
simple-query-server/
├── cmd/server/main.go           # Main entry point - CLI flag handling and server startup
├── internal/config/loader.go    # YAML configuration loading and validation
├── internal/query/executor.go   # Query execution engine (PostgreSQL only)
├── internal/server/http.go      # HTTP server and REST API routing
├── example/sql/schema.sql       # PostgreSQL database schema
├── example/sql/data.sql         # Sample data for PostgreSQL
├── example/                     # Example configuration files (PostgreSQL)
├── testdata/                    # Test configuration files
├── docker-compose.yml           # PostgreSQL database setup
└── go.mod                       # Go module definition with PostgreSQL driver
```

## Key Implementation Details

- **PostgreSQL Support**: Full PostgreSQL database integration with real query execution - **database connection required**
- **Parameter Binding**: Converts `:param` syntax to PostgreSQL `$1, $2...` parameter binding
- **Database Requirement**: Server fails immediately if database connection is unavailable (no fallback)
- **Parameter Validation**: Automatic validation of query parameters with type checking
- **Error Handling**: Comprehensive error responses for missing configs, invalid queries, database errors, etc.
- **Docker Integration**: Complete PostgreSQL setup with docker-compose, schema, and sample data
- **No Tests**: Currently no automated tests exist - manual API validation is required

## Development Workflow

1. **Build and validate changes:**

   ```bash
   make all
   # OR: go build -o server ./cmd/server && go vet ./...
   ```

2. **Test your changes:**

   - Start database: `docker compose up -d postgres`
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
- **Database connection failed / Server exits immediately**: Ensure PostgreSQL is running (`docker compose up -d postgres`) and connection details are correct - **server requires database connection to start**
- **Query not found**: Verify query name matches exactly what's defined in queries.yaml
- **Parameter validation error**: Check parameter names and types match query definition
- **Build failures**: Ensure Go 1.18+ is installed and go.mod is clean

## Important Notes

- NEVER CANCEL builds - first build takes ~11 seconds, subsequent builds <1 second
- Always test with actual API calls after making changes - simply starting/stopping the server is insufficient
- ALWAYS run through complete end-to-end validation scenarios after making changes
- **PostgreSQL database MUST be running** - server will exit with error if database connection fails
- PostgreSQL database includes 23 sample users with various statuses for testing
- Configuration files must be valid YAML with proper structure
- Parameter binding uses `:parameter_name` syntax in SQL queries (converted to PostgreSQL `$1, $2...` format)
- Clean builds require downloading dependencies: `go clean -cache -modcache` before building
