# Monitor Agent

A comprehensive Golang application for monitoring bug bounty programs from multiple platforms (HackerOne, BugCrowd) and discovering their in-scope assets using Project Discovery's ChaosDB.

## Features

- **Multi-Platform Support**: Integrates with HackerOne and BugCrowd APIs
- **Asset Discovery**: Uses ChaosDB to discover additional subdomains and assets
- **Database Storage**: PostgreSQL database for persistent storage
- **Rate Limiting**: Built-in rate limiting to respect API limits
- **One-off Scanning**: Performs single scans with full state management
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
- **Input Validation**: Comprehensive input validation and sanitization

### Performance & Scalability
- **Worker Pools**: Concurrent processing for improved performance
- **Connection Pooling**: Optimized database connection management
- **Rate Limiting**: Intelligent rate limiting with exponential backoff
- **Memory Management**: Efficient memory usage with monitoring

## Architecture

```
Monitor Agent
├── cmd/monitor-agent/     # Application entry point
├── internal/
│   ├── config/           # Configuration management
│   ├── database/         # Database layer and repositories
│   ├── discovery/        # Asset discovery (ChaosDB)
│   ├── metrics/          # Prometheus metrics
│   ├── platforms/        # Platform integrations (HackerOne, BugCrowd)
│   ├── service/          # Business logic layer
│   └── utils/            # Utilities (URL processing, logging, etc.)
├── tests/                # Integration tests
└── docker/               # Docker configuration
```

## Quick Start

### Prerequisites
- Go 1.21 or later
- PostgreSQL database
- API keys for HackerOne, BugCrowd, and ChaosDB (optional - application will only scan platforms with configured keys)

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd Monitor-Agent
   ```

2. **Set up environment variables**
   ```bash
   cp env.example .env
   # Edit .env with your configuration
   ```

3. **Run the application**
   ```bash
   # Run a scan (default behavior)
   go run cmd/monitor-agent/main.go
   
   # Or explicitly run a scan
   go run cmd/monitor-agent/main.go scan
   
   # Show statistics
   go run cmd/monitor-agent/main.go stats
   
   # Health check
   go run cmd/monitor-agent/main.go health
   ```

### Docker Deployment

1. **Build the image**
   ```bash
   docker build -f docker/Dockerfile -t monitor-agent .
   ```

2. **Run with Docker Compose**
   ```bash
   cd docker
   docker-compose up -d
   ```

3. **Run as one-off container**
   ```bash
   docker run --env-file .env monitor-agent
   ```

## Configuration

### Environment Variables

#### Database Configuration
- `DB_HOST`: PostgreSQL host
- `DB_PORT`: PostgreSQL port (default: 5432)
- `DB_NAME`: Database name
- `DB_USER`: Database user
- `DB_PASSWORD`: Database password
- `DB_SSL_MODE`: SSL mode (disable, require, verify-ca, verify-full)
- `DB_SSL_CERT`, `DB_SSL_KEY`, `DB_SSL_ROOT_CERT`: SSL certificates
- `DB_CONNECT_TIMEOUT`: Connection timeout
- `DB_MAX_OPEN_CONNS`: Maximum open connections
- `DB_MAX_IDLE_CONNS`: Maximum idle connections
- `DB_CONN_MAX_LIFETIME`: Connection max lifetime

#### API Configuration
- `HACKERONE_USERNAME`: HackerOne username (required with API key)
- `HACKERONE_API_KEY`: HackerOne API key (optional)
- `BUGCROWD_API_KEY`: BugCrowd API key (optional)
- `CHAOSDB_API_KEY`: ChaosDB API key (optional)
- `HACKERONE_RATE_LIMIT`: HackerOne rate limit (default: 550)
- `BUGCROWD_RATE_LIMIT`: BugCrowd rate limit (default: 55)
- `CHAOSDB_RATE_LIMIT`: ChaosDB rate limit (default: 55)

#### Application Configuration
- `LOG_LEVEL`: Log level (debug, info, warn, error, fatal)
- `ENVIRONMENT`: Environment (development, staging, production)

#### HTTP Configuration
- `HTTP_TIMEOUT`: HTTP timeout
- `HTTP_RETRY_ATTEMPTS`: Number of retry attempts
- `HTTP_RETRY_DELAY`: Retry delay

#### Discovery Configuration
- `CHAOSDB_BULK_SIZE`: Bulk size for ChaosDB requests

#### HTTPX Probe Configuration
- `HTTPX_ENABLED`: Enable HTTPX probe for filtering ChaosDB results (default: true)
- `HTTPX_TIMEOUT`: HTTPX probe timeout per domain (default: 15s)
- `HTTPX_CONCURRENCY`: Number of concurrent HTTPX probes (default: 25)
- `HTTPX_RATE_LIMIT`: HTTPX probe rate limit (default: 100)
- `HTTPX_FOLLOW_REDIRECTS`: Follow HTTP redirects (default: true)
- `HTTPX_MAX_REDIRECTS`: Maximum number of redirects to follow (default: 3)

#### Program-Level Timeouts
- `PROGRAM_PROCESS_TIMEOUT`: Maximum time to process a single program (default: 45m)
- `CHAOS_DISCOVERY_TIMEOUT`: Maximum time for ChaosDB discovery and HTTPX probing (default: 30m)

**Note**: API keys are optional. The application will only scan platforms that have valid API keys configured. If no API keys are provided, the application will start but cannot perform scans.

#### Advanced Configuration
- `CIRCUIT_BREAKER_*`: Circuit breaker settings
- `WORKER_POOL_*`: Worker pool configuration
- `METRICS_*`: Prometheus metrics settings
- `LOG_*`: Logging configuration
- `HEALTH_CHECK_*`: Health check settings
- `API_KEY_ROTATION_*`: API key rotation settings

## Usage

### Commands

- **`monitor-agent`** or **`monitor-agent scan`**: Perform a scan of all platforms
- **`monitor-agent stats`**: Show program and asset statistics
- **`monitor-agent health`**: Perform health checks
- **`monitor-agent help`**: Show help information

### Scheduling

Since this application performs one-off scans, you can schedule it using:

1. **System Cron**:
   ```bash
   # Add to crontab
   0 */6 * * * /path/to/monitor-agent
   ```

2. **Kubernetes CronJob**:
   ```yaml
   apiVersion: batch/v1
   kind: CronJob
   metadata:
     name: monitor-agent
   spec:
     schedule: "0 */6 * * *"
     jobTemplate:
       spec:
         template:
           spec:
             containers:
             - name: monitor-agent
               image: monitor-agent:latest
             restartPolicy: OnFailure
   ```

3. **Docker with external cron**:
   ```bash
   # Run every 6 hours
   0 */6 * * * docker run --env-file .env monitor-agent
   ```

## API Integration

### HackerOne
- Fetches public programs and their scope
- Rate limited to 600 requests per minute
- Requires both username and API key for authentication
- Uses Basic Authentication with username:api_key format

### BugCrowd
- Fetches public programs and their scope
- Rate limited to 60 requests per minute per IP
- Supports API key authentication

### ChaosDB
- Discovers additional subdomains for domains in scope
- Rate limited to 60 requests per minute per IP
- Supports API key authentication for higher limits
- **HTTPX Probe Integration**: Automatically filters out non-existent domains using Project Discovery's HTTPX library
  - Configurable timeout, concurrency, and rate limiting
  - Follows redirects and handles various HTTP status codes
  - Only saves domains that actually exist and respond to HTTP requests
  - **Robust Timeout Handling**: 15-second per-domain timeout with graceful fallback
  - **Crash Prevention**: Comprehensive panic recovery and error handling

## Database Schema

The application uses PostgreSQL with the following main tables:

- **programs**: Bug bounty programs from various platforms
- **assets**: In-scope assets (domains, subdomains, URLs)
- **scans**: Scan history and results

## Test Coverage

Run the test suite:

```bash
# Unit tests
go test ./...

# Integration tests
go test ./tests/integration/...

# With coverage
go test -cover ./...
```

## Development

### Project Structure
- **Clean Architecture**: Clear separation of concerns
- **Repository Pattern**: Database abstraction layer
- **Dependency Injection**: Easy testing and configuration
- **Error Handling**: Comprehensive error handling throughout

### Development Patterns
- **Structured Logging**: JSON logs with correlation IDs
- **Metrics Collection**: Prometheus metrics for monitoring
- **Health Checks**: Comprehensive health monitoring
- **Circuit Breakers**: Fault tolerance for external APIs

## Deployment

### Production Checklist

- [ ] Set up PostgreSQL database with proper SSL configuration
- [ ] Configure API keys for all platforms
- [ ] Set appropriate rate limits
- [ ] Configure logging and monitoring
- [ ] Set up external scheduling (cron, Kubernetes CronJob)
- [ ] Configure health checks and alerting
- [ ] Set up backup and recovery procedures

### Monitoring

The application exposes Prometheus metrics at `/metrics` and provides health check endpoints. Monitor:

- Scan success/failure rates
- API response times and error rates
- Database connection pool status
- Memory and CPU usage
- Asset discovery rates

## Roadmap

- [ ] Support for additional bug bounty platforms
- [ ] Advanced asset discovery techniques
- [ ] Real-time notifications
- [ ] Web dashboard for monitoring
- [ ] Advanced filtering and search capabilities
- [ ] Integration with vulnerability scanners
- [ ] Automated reporting and analytics

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

[Add your license information here]
