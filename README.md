# Monitor Agent

A comprehensive Golang application for monitoring bug bounty programs from multiple platforms (HackerOne, BugCrowd) and discovering their in-scope assets using Project Discovery's ChaosDB.

## Features

- **Multi-Platform Support**: Integrates with HackerOne and BugCrowd APIs
- **Asset Discovery**: Uses ChaosDB to discover additional subdomains and assets
- **Database Storage**: PostgreSQL database for persistent storage
- **Rate Limiting**: Built-in rate limiting to respect API limits
- **Cron Scheduling**: Automated scanning with configurable schedules
- **Docker Support**: Complete containerization with Docker and Docker Compose
- **Comprehensive Testing**: Unit, integration, and end-to-end tests
- **Health Monitoring**: Built-in health checks and monitoring
- **Production Ready**: Circuit breakers, structured logging, metrics, and error handling
- **API Key Rotation**: Support for multiple API keys with automatic rotation
- **Worker Pools**: Concurrent processing for improved performance
- **Configuration Validation**: Comprehensive validation of all configuration parameters

## Production Features

### Error Handling & Resilience
- **Graceful Error Handling**: Proper error handling without application crashes
- **Circuit Breaker Pattern**: Protects against cascading failures from external APIs
- **Health Checks**: Comprehensive health monitoring for all components
- **Recovery Mechanisms**: Automatic recovery from transient failures

### Monitoring & Observability
- **Structured Logging**: JSON-formatted logs with correlation IDs
- **Prometheus Metrics**: Comprehensive metrics for monitoring and alerting
- **Health Endpoints**: Built-in health check endpoints
- **Performance Monitoring**: Memory usage, goroutine count, and response times

### Security & Configuration
- **API Key Rotation**: Support for multiple API keys with automatic rotation
- **Configuration Validation**: Comprehensive validation of all settings
- **SSL/TLS Support**: Full SSL certificate support for database connections
- **Input Validation**: Proper validation and sanitization of all inputs

### Performance & Scalability
- **Worker Pools**: Concurrent processing for improved throughput
- **Connection Pooling**: Optimized database connection management
- **Rate Limiting**: Intelligent rate limiting with exponential backoff
- **Memory Management**: Efficient memory usage with monitoring

## Architecture

The application follows clean architecture principles with the following structure:

```
Monitor-Agent/
├── cmd/monitor-agent/          # Application entry point
├── internal/
│   ├── config/                 # Configuration management
│   ├── database/               # Database models and repositories
│   ├── platforms/              # Bug bounty platform integrations
│   ├── discovery/              # Asset discovery services
│   ├── service/                # Business logic and orchestration
│   ├── utils/                  # Utility functions
│   └── metrics/                # Prometheus metrics
├── tests/                      # Test suites
├── docker/                     # Docker configuration
└── docs/                       # Documentation
```

## Prerequisites

- Go 1.21 or later
- PostgreSQL 12 or later
- Docker and Docker Compose (for containerized deployment)
- API keys for:
  - HackerOne
  - BugCrowd
  - Project Discovery ChaosDB

## Installation

### Local Development

1. **Clone the repository**:
   ```bash
   git clone https://github.com/your-username/Monitor-Agent.git
   cd Monitor-Agent
   ```

2. **Install dependencies**:
   ```bash
   make install-deps
   go mod download
   ```

3. **Set up environment variables**:
   ```bash
   cp env.example .env
   # Edit .env with your API keys and database configuration
   ```

4. **Set up database**:
   ```bash
   # Configure your remote PostgreSQL connection in .env file
   # The application now uses remote PostgreSQL instead of local Docker
   
   # Run database migrations
   make setup-db
   
   # Or use the migration script directly
   ./scripts/migrate.sh --all
   ```

5. **Build and run**:
   ```bash
   make build
   make run
   ```

### Docker Deployment

1. **Set up environment variables**:
   ```bash
   cp env.example .env
   # Edit .env with your remote PostgreSQL configuration
   ```

2. **Start the application**:
   ```bash
   make docker-compose-up
   ```

3. **View logs**:
   ```bash
   make docker-compose-logs
   ```

**Note**: The Docker deployment uses your remote PostgreSQL instance instead of a local container.

## Configuration

The application uses environment variables for configuration. See `env.example` for all available options:

### Required Environment Variables

```bash
# Database - Remote PostgreSQL
DB_HOST=your-remote-postgres-host.com
DB_PORT=5432
DB_NAME=monitor_agent
DB_USER=monitor_agent
DB_PASSWORD=your_secure_password
DB_SSL_MODE=require
DB_SSL_CERT=
DB_SSL_KEY=
DB_SSL_ROOT_CERT=

# API Keys
HACKERONE_API_KEY=your_hackerone_api_key
BUGCROWD_API_KEY=your_bugcrowd_api_key
CHAOSDB_API_KEY=your_chaosdb_api_key
```

### Optional Configuration

```bash
# Database Connection Pool
DB_CONNECT_TIMEOUT=30s
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# SSL Configuration (if using SSL certificates)
DB_SSL_CERT=
DB_SSL_KEY=
DB_SSL_ROOT_CERT=

# Rate Limiting (Optional - defaults are set to be just under API limits)
# HackerOne: 600 requests per minute (default: 550)
# BugCrowd: 60 requests per minute per IP (default: 55)
# ChaosDB: 60 requests per minute per IP (default: 55)
HACKERONE_RATE_LIMIT=550
BUGCROWD_RATE_LIMIT=55
CHAOSDB_RATE_LIMIT=55

# Application
LOG_LEVEL=info
ENVIRONMENT=production
CRON_SCHEDULE="0 */6 * * *"  # Every 6 hours

# HTTP Client
HTTP_TIMEOUT=30s
HTTP_RETRY_ATTEMPTS=3
HTTP_RETRY_DELAY=1s

# Discovery Configuration
CHAOSDB_BULK_SIZE=100
DISCOVERY_CONCURRENT_LIMIT=10

# Circuit Breaker Configuration
CIRCUIT_BREAKER_FAILURE_THRESHOLD=5
CIRCUIT_BREAKER_RECOVERY_TIMEOUT=60s
CIRCUIT_BREAKER_SUCCESS_THRESHOLD=3
```

### Remote Database Configuration

The application is designed to work with remote PostgreSQL instances. Key considerations:

- **SSL Mode**: Set `DB_SSL_MODE` to `require` or `verify-full` for secure connections
- **Connection Pool**: Configure connection pool settings for optimal performance
- **Network Latency**: Consider increasing `DB_CONNECT_TIMEOUT` for remote connections
- **SSL Certificates**: For `verify-full` SSL mode, provide certificate paths in `DB_SSL_CERT`, `DB_SSL_KEY`, and `DB_SSL_ROOT_CERT`

For detailed setup instructions, see [Remote Database Setup Guide](docs/REMOTE_DATABASE_SETUP.md).

## Usage

### Command Line Interface

The application provides several commands:

```bash
# Run as a service with cron scheduling
./monitor-agent

# Perform a single scan
./monitor-agent scan

# Show statistics
./monitor-agent stats

# Run health checks
./monitor-agent health

# Show help
./monitor-agent help
```

### Using Make Commands

```bash
# Build the application
make build

# Run tests
make test

# Run with Docker
make docker-compose-up

# Show logs
make docker-compose-logs

# Database management
make test-db-connection  # Test database connection
make migrate-validate    # Validate database connection
make migrate             # Run migrations only
make migrate-verify      # Verify migration results
```

## API Integration

### HackerOne

The application integrates with HackerOne's API to:
- Retrieve public bug bounty programs
- Get in-scope assets for each program
- Handle rate limiting and pagination
- Circuit breaker protection for API failures

### BugCrowd

Similar integration for BugCrowd's API with:
- Full API integration (not just in progress)
- Rate limiting and error handling
- Circuit breaker protection

### ChaosDB

Uses Project Discovery's ChaosDB to:
- Discover additional subdomains for known domains
- Bulk discovery for efficiency
- Rate limiting to respect API limits
- Error handling and retry logic

## Database Schema

The application uses PostgreSQL with the following main tables:

- **programs**: Bug bounty programs from various platforms
- **assets**: Discovered assets (subdomains, URLs)
- **asset_responses**: HTTP response information for assets
- **scans**: Discovery scan sessions
- **platforms**: Bug bounty platform information

## Testing

The application includes comprehensive testing:

```bash
# Run unit tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration

# Run end-to-end tests
make test-e2e

# Run all tests
make all
```

### Test Coverage

- **Unit Tests**: Individual component testing
- **Integration Tests**: Database and API integration testing
- **Configuration Tests**: Validation and error handling
- **Performance Tests**: Load testing and memory usage
- **Error Recovery Tests**: Failure scenario testing

## Development

### Project Structure

- **Clean Architecture**: Separation of concerns with clear boundaries
- **Dependency Injection**: Easy testing and modularity
- **Repository Pattern**: Database abstraction
- **Factory Pattern**: Platform client creation
- **Strategy Pattern**: Different discovery strategies
- **Circuit Breaker Pattern**: Fault tolerance
- **Worker Pool Pattern**: Concurrent processing

### Adding New Platforms

To add support for a new bug bounty platform:

1. Implement the `Platform` interface in `internal/platforms/`
2. Add platform configuration to the factory
3. Update tests and documentation
4. Add circuit breaker configuration

### Adding New Discovery Services

To add new asset discovery services:

1. Implement discovery logic in `internal/discovery/`
2. Integrate with the monitor service
3. Add configuration options
4. Add metrics and monitoring

## Deployment

### Production Deployment

1. **Set up PostgreSQL database**
2. **Configure environment variables**
3. **Set up monitoring and alerting**
4. **Build and deploy**:
   ```bash
   make release
   docker push your-registry/monitor-agent:latest
   ```

### VPS Deployment

1. **Install Docker and Docker Compose**
2. **Clone and configure the application**
3. **Set up SSL certificates**
4. **Start services**:
   ```bash
   make docker-compose-up
   ```

### Monitoring and Logging

- **Application logs**: JSON-formatted logs with correlation IDs
- **Health checks**: Via the `/health` endpoint
- **Prometheus metrics**: Available on `/metrics` endpoint
- **Database monitoring**: Through standard PostgreSQL tools
- **Circuit breaker status**: Available in metrics

### Production Checklist

- [ ] SSL certificates configured
- [ ] Database connection pool optimized
- [ ] Rate limits configured
- [ ] Circuit breakers enabled
- [ ] Monitoring and alerting set up
- [ ] Log aggregation configured
- [ ] Backup strategy implemented
- [ ] Security scanning completed

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Create an issue on GitHub
- Check the documentation in the `docs/` directory
- Review the test examples for usage patterns

## Roadmap

- [x] BugCrowd API integration (completed)
- [x] Circuit breaker pattern (completed)
- [x] Structured logging (completed)
- [x] Prometheus metrics (completed)
- [x] API key rotation (completed)
- [x] Worker pools (completed)
- [x] Configuration validation (completed)
- [ ] Additional bug bounty platforms
- [ ] Web interface for monitoring
- [ ] Advanced asset filtering
- [ ] Integration with security tools
- [ ] Automated vulnerability scanning
- [ ] Slack/Discord notifications
- [ ] Metrics and analytics dashboard
