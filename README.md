# simple-query-server

A lightweight Go server with YAML-based configuration for database connections and query definitions.

## Overview

The `simple-query-server` allows you to define database queries in YAML configuration files and expose them as HTTP REST endpoints. This provides a simple way to create database APIs without writing custom code for each query.

## Features

- **YAML Configuration**: Define database connections and queries in separate YAML files
- **HTTP API**: Execute queries via REST endpoints with JSON payloads
- **Parameter Validation**: Automatic validation of query parameters
- **PostgreSQL Support**: Full PostgreSQL database support with connection pooling
- **Docker Integration**: Complete PostgreSQL setup with docker-compose
- **Command Line Interface**: Flexible configuration via CLI flags

## Project Structure

```
simple-query-server/
├── cmd/
│   └── server/
│       └── main.go           # Main entry point
├── example/
│   ├── database.yaml         # Example database configuration (PostgreSQL)
│   ├── queries.yaml          # Example queries configuration
│   └── sql/
│       ├── schema.sql        # Database schema (tables, indexes, functions)
│       └── data.sql          # Sample data for testing
├── internal/
│   ├── config/
│   │   └── loader.go         # YAML configuration loading
│   ├── query/
│   │   └── executor.go       # Query execution engine (PostgreSQL)
│   └── server/
│       └── http.go           # HTTP server and routing
├── testdata/
│   ├── database.yaml         # Test database configuration
│   └── queries.yaml          # Test queries configuration
├── docker-compose.yml        # PostgreSQL database setup
├── go.mod
├── go.sum
└── README.md
```

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

For PostgreSQL (recommended):

```yaml
type: "postgres"
dsn: "postgres://queryuser:querypass@localhost:5432/queryserver?sslmode=disable"
```

For other databases (future support):

```yaml
# MySQL (not yet implemented)
type: "mysql" 
dsn: "username:password@tcp(localhost:3306)/database_name"

# SQLite (not yet implemented)
type: "sqlite"
dsn: "./data.db"
```

**Note**: Currently only PostgreSQL is supported. Database connection is required - the server will fail to start if no valid database connection is available.

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

With PostgreSQL (recommended):

```bash
# Start PostgreSQL database first
docker compose up -d postgres

# Start the server
./server --db-config ./example/database.yaml --queries-config ./example/queries.yaml
```

Or using make commands:

```bash
make run      # Uses example configuration
make run-test # Uses test configuration on port 8081
```

**Options:**
- `--db-config`: Path to database configuration YAML file (required)
- `--queries-config`: Path to queries configuration YAML file (required)  
- `--port`: Port to run the server on (default: 8080)
- `--help`: Show help message

**Database Connection**: The server requires a valid database connection. If the connection fails, the server will exit with an error message.

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

3. **List available queries**:
   ```bash
   curl http://localhost:8080/queries
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

## Development Status

**Current Implementation:**
- ✅ YAML configuration loading and validation
- ✅ HTTP server with REST API endpoints
- ✅ Parameter validation and type checking
- ✅ **PostgreSQL database connection and query execution**
- ✅ **SQL parameter binding with :param syntax**
- ✅ **Docker Compose setup with sample database**
- ✅ Command-line interface with flags
- ✅ Error handling and logging

**TODO (for production use):**
- [ ] MySQL and SQLite database support
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