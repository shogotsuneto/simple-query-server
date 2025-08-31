# simple-query-server

A lightweight Go server with YAML-based configuration for database connections and query definitions.

## Overview

The `simple-query-server` allows you to define database queries in YAML configuration files and expose them as HTTP REST endpoints. This provides a simple way to create database APIs without writing custom code for each query.

**READ-ONLY OPERATIONS**: This project supports only SELECT (read) operations. No INSERT, UPDATE, DELETE, or other write operations are supported.

## Features

- **READ-ONLY Database Access**: Supports only SELECT queries for safe, read-only database operations
- **YAML Configuration**: Define database connections and queries in separate YAML files
- **HTTP API**: Execute queries via REST endpoints with JSON payloads
- **Parameter Validation**: Automatic validation of query parameters
- **PostgreSQL Support**: Full PostgreSQL database support with background connection management
- **Background Connection Management**: Server starts successfully even when database is unavailable
- **Automatic Reconnection**: Exponential backoff retry mechanism with health monitoring
- **Middleware System**: Configurable middleware for authentication and parameter injection
- **Docker Integration**: Complete PostgreSQL setup with docker-compose
- **Command Line Interface**: Flexible configuration via CLI flags

## Installation and Build

### Prerequisites
- Go 1.18 or later
- Docker and Docker Compose (for PostgreSQL database)

### Quick Start with PostgreSQL

1. **Clone the repository**:
   ```bash
   git clone https://github.com/shogotsuneto/simple-query-server
   cd simple-query-server
   ```

2. **Start PostgreSQL database**:
   ```bash
   docker compose up -d postgres
   ```

3. **Build and start the server**:
   ```bash
   make build
   make run
   ```

4. **Test the API**:
   ```bash
   make api-test
   ```

### Manual Build

If you prefer to build manually without make:

```bash
go build -o server ./cmd/server
./server --db-config ./example/database.yaml --queries-config ./example/queries.yaml --port 8080
```

## Configuration

The server requires two YAML configuration files to run:

### Database Configuration (`database.yaml`)

```yaml
type: "postgres"
dsn: "postgres://queryuser:querypass@localhost:5432/queryserver?sslmode=disable"
```

**Note**: Currently only PostgreSQL is supported. The server starts successfully even when the database is unavailable, with background connection management and automatic reconnection.

### Queries Configuration (`queries.yaml`)

```yaml
queries:
  get_user_by_id:
    sql: "SELECT id, name, email FROM users WHERE id = :id"
    params:
      - name: id
        type: int

  search_users:
    sql: "SELECT id, name FROM users WHERE name LIKE :name"
    params:
      - name: name
        type: string
```

### Middleware Configuration

The server supports optional middleware for request processing, authentication, and parameter injection. See [MIDDLEWARE.md](MIDDLEWARE.md) for detailed configuration and usage documentation.

## Usage

### Starting the Server

With PostgreSQL (recommended):

```bash
# Start PostgreSQL database first (optional - server starts without database)
docker compose up -d postgres

# Start the server with optional middleware support
./server --db-config ./example/database.yaml \
         --queries-config ./example/queries.yaml \
         --server-config ./example/server.yaml
```

Or using make commands:

```bash
make run      # Uses example configuration
make run-test # Uses test configuration on port 8081
```

**Options:**
- `--db-config`: Path to database configuration YAML file (required)
- `--queries-config`: Path to queries configuration YAML file (required)  
- `--server-config`: Path to server configuration YAML file (optional, for middleware)
- `--port`: Port to run the server on (default: 8080)
- `--help`: Show help message

**Database Connection**: The server starts successfully even when the database is unavailable. Connection attempts happen automatically in the background with retry logic and health monitoring.

### API Endpoints

#### Health Check
```bash
GET /health
```
When PostgreSQL is connected:
Response: `{"database":{"connected":true},"status":"healthy"}`

When PostgreSQL is unavailable:
Response (HTTP 503): `{"database":{"connected":false},"status":"unhealthy"}`

#### List Available Queries
```bash
GET /queries
```

#### Execute a Query
```bash
POST /query/{query_name}
Content-Type: application/json

{
  "param1": "value1",
  "param2": "value2"
}
```

### Example API Calls

1. **Get user by ID**:
   ```bash
   curl -X POST -H "Content-Type: application/json" \
        -d '{"id": 1}' \
        http://localhost:8080/query/get_user_by_id
   ```

2. **Search users by name**:
   ```bash
   curl -X POST -H "Content-Type: application/json" \
        -d '{"name": "%Alice%"}' \
        http://localhost:8080/query/search_users
   ```

3. **Get all active users**:
   ```bash
   curl -X POST -H "Content-Type: application/json" \
        -d '{}' \
        http://localhost:8080/query/get_all_active_users
   ```

4. **List users with pagination**:
   ```bash
   curl -X POST -H "Content-Type: application/json" \
        -d '{"limit": 5, "offset": 0}' \
        http://localhost:8080/query/list_users
   ```

5. **Comprehensive API testing**:
   ```bash
   make api-test
   ```

### Response Format

**Success Response:**
```json
{
  "rows": [
    {
      "id": 1,
      "name": "Alice Smith",
      "email": "alice.smith@example.com"
    }
  ]
}
```

**Empty Result Response:**
```json
{}
```

**Error Response:**
```json
{
  "error": "Query 'invalid_query' not found"
}
```

## Testing

### Manual API Testing

For quick validation with a running server:

```bash
make api-test
```

This runs comprehensive curl-based tests against the running server.

### Integration Tests

For automated testing with full environment setup:

```bash
make integration-test
```

This command:
- Starts an isolated PostgreSQL database in Docker
- Builds and starts the server with test configuration
- Runs comprehensive Go-based integration tests
- Tests all API endpoints with real HTTP requests
- Validates error handling and edge cases
- Automatically cleans up all test resources

See [integration/README.md](integration/README.md) for detailed integration testing documentation.

## Development Status

**Current Implementation:**
- ✅ PostgreSQL database support with background connection management
- ✅ YAML-based configuration for database connections and queries  
- ✅ REST API endpoints with parameter validation
- ✅ Middleware system with HTTP header and JWT/JWKS authentication
- ✅ Docker integration and integration tests

**Future Enhancements:**
- [ ] MySQL and SQLite database support
- [ ] Database connection pooling configuration
- [ ] Query result caching
- [ ] Rate limiting and request logging middleware
- [ ] Metrics and monitoring

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.