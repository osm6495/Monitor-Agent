# Migration to Remote PostgreSQL - Summary

This document summarizes the changes made to migrate the Monitor Agent from using a local PostgreSQL Docker container to a remote PostgreSQL instance.

## Overview

The application has been updated to support remote PostgreSQL connections with enhanced SSL support, connection pooling, and improved configuration management.

## Key Changes

### 1. Configuration Updates

#### Environment Variables (`env.example`)
- **Added**: Remote database configuration parameters
  - `DB_SSL_CERT`, `DB_SSL_KEY`, `DB_SSL_ROOT_CERT` for SSL certificates
  - `DB_CONNECT_TIMEOUT` for connection timeout
  - `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME` for connection pooling
- **Changed**: Default `DB_SSL_MODE` from `disable` to `require`
- **Changed**: Default `DB_HOST` from `localhost` to `your-remote-postgres-host.com`

#### Configuration Code (`internal/config/config.go`)
- **Enhanced**: `DatabaseConfig` struct with new SSL and connection pool fields
- **Updated**: `Load()` function to parse new environment variables
- **Improved**: `GetDSN()` method to include SSL certificate parameters and connection timeout

### 2. Database Connection Updates

#### Main Application (`cmd/monitor-agent/main.go`)
- **Updated**: `connectToDatabase()` function to use configuration-based connection pool settings
- **Removed**: Hardcoded connection pool values

### 3. Docker Configuration

#### Production Docker Compose (`docker/docker-compose.yml`)
- **Removed**: Local PostgreSQL service
- **Updated**: Monitor agent service to use remote database configuration
- **Removed**: Dependencies on local PostgreSQL service
- **Removed**: Local PostgreSQL volumes and networks



### 4. Scripts and Tools

#### Migration Script (`scripts/migrate.sh`)
- **Created**: Comprehensive migration script for remote database setup
- **Features**: Connection validation, migration execution, and verification
- **Options**: `--validate`, `--migrate`, `--verify`, `--all`

#### Connection Test Script (`scripts/test-db-connection.sh`)
- **Created**: Database connection testing tool
- **Features**: Network connectivity, database connection, and SSL testing
- **Options**: `--network`, `--database`, `--ssl`, `--all`

#### Setup Script (`scripts/setup.sh`)
- **Updated**: Database setup function to guide users to remote configuration
- **Removed**: Local Docker PostgreSQL setup instructions

### 5. Makefile Updates

#### New Commands
- `make test-db-connection` - Test database connection
- `make migrate` - Run database migrations
- `make migrate-validate` - Validate database connection
- `make migrate-verify` - Verify migration results

#### Updated Commands
- `make setup-db` - Now uses migration script for remote database setup

### 6. Documentation

#### README.md
- **Updated**: Installation instructions for remote database
- **Added**: Remote database configuration section
- **Updated**: Docker deployment notes
- **Added**: Database management commands
- **Added**: Reference to detailed setup guide

#### Remote Database Setup Guide (`docs/REMOTE_DATABASE_SETUP.md`)
- **Created**: Comprehensive guide for remote PostgreSQL setup
- **Includes**: Configuration steps, SSL setup, troubleshooting, and provider-specific notes

## Rate Limit Configuration Updates

### Default Rate Limits
The application now uses sensible default rate limits that are just under the actual API limits:

- **HackerOne**: 550 requests per minute (API limit: 600 requests per minute)
- **BugCrowd**: 55 requests per minute per IP (API limit: 60 requests per minute per IP)
- **ChaosDB**: 55 requests per minute per IP (API limit: 60 requests per minute per IP)

### Environment Variables
Rate limit environment variables are now **optional** and will use the safe defaults if not provided:

```bash
# Optional - will use defaults if not set
HACKERONE_RATE_LIMIT=550
BUGCROWD_RATE_LIMIT=55
CHAOSDB_RATE_LIMIT=55
```

### Configuration Files Updated
- `internal/config/config.go` - Updated default values
- `env.example` - Updated with new defaults and documentation
- `docker/docker-compose.yml` - Updated default values
- `README.md` - Updated documentation
- `internal/config/config_test.go` - Added test for default values

## Migration Steps for Users

### 1. Update Environment Configuration
```bash
cp env.example .env
# Edit .env with your remote PostgreSQL configuration
# Rate limit variables are now optional with safe defaults
```

### 2. Test Database Connection
```bash
make test-db-connection
```

### 3. Run Database Migrations
```bash
make setup-db
```

### 4. Start Application
```bash
# Start the application
make docker-compose-up
```

## Benefits of Remote Database

### 1. Scalability
- No local resource constraints
- Better performance for production workloads
- Easier horizontal scaling

### 2. Reliability
- Managed database services provide high availability
- Automatic backups and maintenance
- Better disaster recovery

### 3. Security
- Enhanced SSL/TLS support
- Network-level security controls
- Managed security updates

### 4. Management
- Reduced operational overhead
- Centralized database management
- Better monitoring and alerting

## Backward Compatibility

The application maintains backward compatibility with local PostgreSQL instances by:
- Supporting `DB_SSL_MODE=disable` for local connections
- Maintaining the same database schema
- Preserving all existing functionality

## Testing

To test the migration:

1. **Connection Testing**: Use `make test-db-connection`
2. **Migration Testing**: Use `make migrate-validate` and `make migrate-verify`
3. **Application Testing**: Use `make run` or `make docker-compose-up`

## Support

For issues with remote database setup:
1. Check the [Remote Database Setup Guide](docs/REMOTE_DATABASE_SETUP.md)
2. Use the provided test scripts
3. Review the troubleshooting section in the setup guide
4. Check application logs for detailed error messages
