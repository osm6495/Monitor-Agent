# Remote Database Setup Guide

This guide will help you configure the Monitor Agent to use a remote PostgreSQL instance instead of a local database.

## Prerequisites

- A remote PostgreSQL instance (e.g., AWS RDS, Google Cloud SQL, DigitalOcean Managed Databases, or your own VPS)
- PostgreSQL client tools installed locally (`psql` command)
- Network access to your PostgreSQL instance

## Configuration Steps

### 1. Environment Configuration

Copy the example environment file and configure your database connection:

```bash
cp env.example .env
```

Edit the `.env` file with your remote PostgreSQL configuration:

```bash
# Database Configuration - Remote PostgreSQL
DB_HOST=your-remote-postgres-host.com
DB_PORT=5432
DB_NAME=monitor_agent
DB_USER=monitor_agent
DB_PASSWORD=your_secure_password
DB_SSL_MODE=require
DB_CONNECT_TIMEOUT=30s
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
```

### 2. SSL Configuration

For secure connections, configure SSL settings based on your provider:

#### AWS RDS
```bash
DB_SSL_MODE=require
# No additional SSL certificates needed for RDS
```

#### Google Cloud SQL
```bash
DB_SSL_MODE=require
# Download and configure SSL certificates if needed
DB_SSL_ROOT_CERT=/path/to/server-ca.pem
DB_SSL_CERT=/path/to/client-cert.pem
DB_SSL_KEY=/path/to/client-key.pem
```

#### DigitalOcean Managed Databases
```bash
DB_SSL_MODE=require
# No additional SSL certificates needed
```

#### Custom PostgreSQL Server
```bash
DB_SSL_MODE=verify-full
DB_SSL_ROOT_CERT=/path/to/ca-certificate.crt
DB_SSL_CERT=/path/to/client-certificate.crt
DB_SSL_KEY=/path/to/client-key.key
```

### 3. Database Setup

Create the database and user on your PostgreSQL instance:

```sql
-- Connect to your PostgreSQL instance as superuser
CREATE DATABASE monitor_agent;
CREATE USER monitor_agent WITH PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE monitor_agent TO monitor_agent;
```

### 4. Run Migrations

Use the migration script to set up the database schema:

```bash
# Test database connection
./scripts/migrate.sh --validate

# Run complete migration (validate, migrate, verify)
./scripts/migrate.sh --all

# Or use make commands
make migrate-validate
make setup-db
```

## Connection Testing

### Test with psql

Test your connection manually:

```bash
psql -h your-remote-postgres-host.com -p 5432 -U monitor_agent -d monitor_agent
```

### Test with Migration Script

```bash
./scripts/migrate.sh --validate
```

### Test with Application

```bash
# Build and test the application
make build
./build/monitor-agent health
```

## Troubleshooting

### Connection Issues

1. **Network Connectivity**
   ```bash
   # Test basic connectivity
   telnet your-remote-postgres-host.com 5432
   
   # Test with nc
   nc -zv your-remote-postgres-host.com 5432
   ```

2. **Firewall Configuration**
   - Ensure your PostgreSQL instance allows connections from your application's IP
   - Check security groups (AWS) or firewall rules

3. **SSL Issues**
   - Verify SSL mode matches your provider's requirements
   - Check certificate paths and permissions
   - Try different SSL modes: `disable`, `require`, `verify-ca`, `verify-full`

### Common Error Messages

#### Connection Refused
```
Error: failed to connect to database: dial tcp: connect: connection refused
```
**Solution**: Check if PostgreSQL is running and accessible on the specified port.

#### Authentication Failed
```
Error: failed to connect to database: pq: password authentication failed
```
**Solution**: Verify username and password in your `.env` file.

#### SSL Connection Failed
```
Error: failed to connect to database: pq: SSL is not enabled on the server
```
**Solution**: Set `DB_SSL_MODE=disable` or configure SSL on your PostgreSQL server.

#### Database Does Not Exist
```
Error: failed to connect to database: pq: database "monitor_agent" does not exist
```
**Solution**: Create the database and user as shown in step 3.

## Performance Optimization

### Connection Pool Settings

Adjust connection pool settings based on your workload:

```bash
# For high-traffic applications
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=10m

# For low-traffic applications
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=2
DB_CONN_MAX_LIFETIME=5m
```

### Network Latency

For remote databases with high latency:

```bash
# Increase connection timeout
DB_CONNECT_TIMEOUT=60s

# Increase HTTP timeouts if needed
HTTP_TIMEOUT=60s
```

## Security Best Practices

1. **Use Strong Passwords**: Generate secure passwords for database users
2. **Enable SSL**: Always use SSL for remote database connections
3. **Network Security**: Restrict database access to specific IP ranges
4. **Regular Updates**: Keep your PostgreSQL instance updated
5. **Backup Strategy**: Implement regular database backups
6. **Monitoring**: Set up monitoring for database performance and security

## Provider-Specific Notes

### AWS RDS
- Use RDS security groups to control access
- Enable encryption at rest
- Consider using IAM database authentication

### Google Cloud SQL
- Use Cloud SQL Proxy for secure connections
- Enable automatic backups
- Configure maintenance windows

### DigitalOcean Managed Databases
- Use trusted sources for network access
- Enable automated backups
- Monitor connection limits

## Support

If you encounter issues:

1. Check the troubleshooting section above
2. Review your provider's documentation
3. Test with a simple `psql` connection
4. Check application logs for detailed error messages
5. Verify all environment variables are correctly set
