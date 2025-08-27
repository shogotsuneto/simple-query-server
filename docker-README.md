# PostgreSQL Database Configuration for Docker Setup

This directory contains the necessary files for running a PostgreSQL database with the simple-query-server.

## Quick Start

1. **Start PostgreSQL with Docker Compose:**
   ```bash
   docker-compose up -d postgres
   ```

2. **Wait for database to be ready:**
   ```bash
   docker-compose logs postgres
   # Wait for "database system is ready to accept connections"
   ```

3. **Build and start the server:**
   ```bash
   make build
   make run
   ```

4. **Test the API:**
   ```bash
   make api-test
   ```

## Files

- `docker-compose.yml` - Docker Compose configuration with PostgreSQL service
- `sql/schema.sql` - Database schema with tables and indexes
- `sql/data.sql` - Sample data for testing
- `example/database.yaml` - Database configuration for connecting to PostgreSQL

## Database Details

- **Host:** localhost
- **Port:** 5432
- **Database:** queryserver
- **Username:** queryuser
- **Password:** querypass

## Optional PgAdmin

To start PgAdmin for database management:

```bash
docker-compose --profile admin up -d
```

Access PgAdmin at http://localhost:5050:
- **Email:** admin@example.com  
- **Password:** admin

## Sample Data

The database is populated with:
- 23 users (18 active, 2 inactive, 3 suspended)
- 14 user profiles with additional information
- Test data for all example queries

## Stopping Services

```bash
docker-compose down
# or to remove volumes as well:
docker-compose down -v
```