# simple-query-server

A lightweight Go server with YAML-based configuration for database connections and query definitions.

## Overview

The `simple-query-server` allows you to define database queries in YAML configuration files and expose them as HTTP REST endpoints. This provides a simple way to create database APIs without writing custom code for each query.

## Features

- **YAML Configuration**: Define database connections and queries in separate YAML files
- **HTTP API**: Execute queries via REST endpoints with JSON payloads
- **Parameter Validation**: Automatic validation of query parameters
- **Multiple Database Support**: Designed to support PostgreSQL, MySQL, and SQLite (with mock responses for now)
- **Command Line Interface**: Flexible configuration via CLI flags

## Project Structure

```
simple-query-server/
├── cmd/
│   └── server/
│       └── main.go           # Main entry point
├── example/
│   ├── database.yaml         # Example database configuration
│   └── queries.yaml          # Example queries configuration
├── internal/
│   ├── config/
│   │   └── loader.go         # YAML configuration loading
│   ├── query/
│   │   └── executor.go       # Query execution engine
│   └── server/
│       └── http.go           # HTTP server and routing
├── testdata/
│   ├── database.yaml         # Test database configuration
│   └── queries.yaml          # Test queries configuration
├── go.mod
├── go.sum
└── README.md
```

## Installation and Build

1. **Prerequisites**: Go 1.18 or later

2. **Clone the repository**:
   ```bash
   git clone https://github.com/shogotsuneto/simple-query-server
   cd simple-query-server
   ```

3. **Build the binary**:
   ```bash
   go build -o server ./cmd/server
   ```

## Configuration

### Database Configuration (`database.yaml`)

```yaml
type: "postgres"  # Supported: postgres, mysql, sqlite
dsn: "postgres://username:password@localhost:5432/database_name?sslmode=disable"

# Optional: separate credentials
credentials:
  username: "your_username"
  password: "your_password"
  host: "localhost"
  port: "5432"
  database: "your_database"
  sslmode: "disable"
```

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

## Usage

### Starting the Server

```bash
./server --db-config ./example/database.yaml --queries-config ./example/queries.yaml
```

**Options:**
- `--db-config`: Path to database configuration YAML file (required)
- `--queries-config`: Path to queries configuration YAML file (required)
- `--port`: Port to run the server on (default: 8080)
- `--help`: Show help message

### API Endpoints

#### Health Check
```bash
GET /health
```
Response: `{"status": "healthy"}`

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
        -d '{"id": 123}' \
        http://localhost:8080/query/get_user_by_id
   ```

2. **Search users by name**:
   ```bash
   curl -X POST -H "Content-Type: application/json" \
        -d '{"name": "%Alice%"}' \
        http://localhost:8080/query/search_users
   ```

3. **List available queries**:
   ```bash
   curl http://localhost:8080/queries
   ```

### Response Format

**Success Response:**
```json
{
  "rows": [
    {
      "id": 123,
      "name": "Alice",
      "email": "alice@example.com"
    }
  ]
}
```

**Error Response:**
```json
{
  "error": "Query 'invalid_query' not found"
}
```

## Development Status

**Current Implementation:**
- ✅ YAML configuration loading and validation
- ✅ HTTP server with REST API endpoints
- ✅ Parameter validation and type checking
- ✅ Mock query responses for demonstration
- ✅ Command-line interface with flags
- ✅ Error handling and logging

**TODO (for production use):**
- [ ] Actual database connection implementation
- [ ] SQL parameter binding (currently using mock responses)
- [ ] Database connection pooling
- [ ] Query result caching
- [ ] Authentication and authorization
- [ ] Rate limiting
- [ ] Metrics and monitoring
- [ ] Integration tests with real databases

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.