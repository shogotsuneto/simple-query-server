# Integration Tests

This directory contains comprehensive integration tests for the simple-query-server that test the complete system with a real database and HTTP server.

## Overview

The integration tests provide:
- **Full system validation** with actual HTTP requests and database interactions
- **Isolated test environment** using dedicated Docker Compose setup
- **Comprehensive test coverage** of all API endpoints and error scenarios
- **Automated setup and teardown** of test infrastructure

## Test Structure

```
integration/
├── README.md                       # This documentation
├── docker-compose.integration.yml  # Docker Compose for test database
├── config/
│   ├── database.yaml              # Integration test database config
│   └── queries.yaml               # Integration test queries config
├── go.mod                         # Go module for integration tests
└── integration_test.go            # Main integration test suite
```

## Running Integration Tests

### Full Integration Test Suite

Run the complete integration test suite with automatic setup and cleanup:

```bash
make integration-test
```

This command will:
1. Build the server binary
2. Start PostgreSQL database in Docker
3. Start the server with test configuration  
4. Run all integration tests
5. Clean up all services automatically

### Manual Setup and Cleanup

For development and debugging, you can manually control the test environment:

```bash
# Start test environment
make integration-test-setup

# Run tests against running environment
cd integration && go test -v

# Clean up test environment
make integration-test-cleanup
```

## Test Coverage

The integration test suite covers:

### API Endpoints
- **Health Check** (`GET /health`) - Service health verification
- **Query Listing** (`GET /queries`) - Available queries enumeration  
- **Query Execution** (`POST /query/{name}`) - Actual query execution with parameters

### Positive Test Scenarios
- User retrieval by ID with valid parameters
- Search functionality with pattern matching
- Pagination queries with limit/offset
- Parameterless queries (get all active users)
- Multi-parameter queries with complex logic
- Status-based aggregation queries

### Error Handling
- Missing required parameters
- Invalid parameter types
- Nonexistent query names
- Invalid SQL execution
- HTTP method restrictions
- Partial parameter sets

### System Integration
- **Database Connectivity** - Real PostgreSQL database operations
- **Parameter Binding** - SQL parameter conversion and validation
- **HTTP Protocol** - Proper status codes, headers, and JSON responses
- **Data Consistency** - Cross-query data verification
- **Concurrent Access** - Multiple test cases with shared database

## Test Environment

### Database Setup
- **PostgreSQL 15** running in Docker container
- **Isolated database** (`queryserver_integration_test`) on port 5433
- **Test user** credentials: `testuser/testpass`
- **Sample data** automatically loaded from example SQL files
- **Clean state** for each test run

### Server Configuration
- **Dedicated port** 8081 to avoid conflicts with development server
- **Test-specific** database and queries configurations
- **Host-based execution** for faster builds and easier debugging

### Network Isolation
- Dedicated Docker network for integration tests
- No interference with development environment
- Clean teardown of all resources

## Test Implementation Details

### Test Structure
- **TestMain** handles setup/teardown of entire test environment
- **Subtests** organize different test scenarios logically
- **Helper functions** for HTTP requests and response validation
- **Comprehensive assertions** for status codes, response format, and data content

### Key Test Cases

#### `TestHealthEndpoint`
Validates the health check endpoint returns proper JSON response.

#### `TestListQueriesEndpoint` 
Verifies all configured queries are properly exposed via the API.

#### `TestQueryExecutionSuccess`
Tests successful execution of all query types with valid parameters.

#### `TestQueryExecutionErrors`
Validates proper error handling for various failure scenarios.

#### `TestHTTPMethods`
Ensures only appropriate HTTP methods are accepted for each endpoint.

#### `TestDataConsistency`
Verifies data consistency across multiple queries using the same database.

## Configuration Files

### `config/database.yaml`
```yaml
type: "postgres"
dsn: "postgres://testuser:testpass@localhost:5433/queryserver_integration_test?sslmode=disable"
```

### `config/queries.yaml`
Contains all standard queries plus additional test-specific queries for error scenarios.

## Best Practices

### When to Run
- **Before commits** to validate changes don't break functionality
- **During development** when modifying API endpoints or query logic  
- **In CI/CD** as part of automated testing pipeline

### Troubleshooting
- Check Docker is running and accessible
- Ensure ports 5433 and 8081 are available
- Verify PostgreSQL container starts successfully
- Check server logs for connection errors

### Extending Tests
- Add new test cases to `integration_test.go`
- Update `config/queries.yaml` for new query patterns
- Follow existing test structure and naming conventions
- Ensure proper cleanup in error scenarios

## Dependencies

- **Go 1.21+** for test execution
- **Docker** for PostgreSQL container
- **Docker Compose** for service orchestration
- **curl** (optional) for manual API testing

The integration tests require no additional Go dependencies beyond those already used by the main application.