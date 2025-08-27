# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Project initiated with core functionality
- YAML configuration loading and validation
- HTTP server with REST API endpoints
- Parameter validation and type checking
- PostgreSQL database connection and query execution
- Background connection management with automatic retry
- Health monitoring with meaningful database status reporting
- SQL parameter binding with :param syntax
- Docker Compose setup with sample database
- Command-line interface with flags
- Error handling and logging
- Integration tests with real databases
- Release pipeline with Docker multi-arch builds

### Docker Usage
Docker images are available for both `linux/amd64` and `linux/arm64` architectures:

```bash
# Pull the latest image
docker pull ghcr.io/shogotsuneto/simple-query-server:latest

# Run with configuration files (mount your config directory)
docker run -d \
  -p 8080:8080 \
  -v /path/to/your/configs:/configs \
  ghcr.io/shogotsuneto/simple-query-server:latest \
  --db-config /configs/database.yaml \
  --queries-config /configs/queries.yaml

# Or use a specific version
docker pull ghcr.io/shogotsuneto/simple-query-server:v1.0.0
```