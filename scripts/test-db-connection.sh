#!/bin/bash

# Monitor Agent Database Connection Test Script

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to load environment variables
load_env() {
    if [ -f ".env" ]; then
        print_status "Loading environment variables from .env file..."
        export $(grep -v '^#' .env | xargs)
    else
        print_warning "No .env file found. Please ensure environment variables are set."
    fi
}

# Function to test network connectivity
test_network() {
    print_status "Testing network connectivity to $DB_HOST:$DB_PORT..."
    
    if command_exists nc; then
        if nc -z -w5 "$DB_HOST" "$DB_PORT" 2>/dev/null; then
            print_success "Network connectivity successful"
            return 0
        else
            print_error "Network connectivity failed"
            return 1
        fi
    elif command_exists telnet; then
        if timeout 5 bash -c "</dev/tcp/$DB_HOST/$DB_PORT" 2>/dev/null; then
            print_success "Network connectivity successful"
            return 0
        else
            print_error "Network connectivity failed"
            return 1
        fi
    else
        print_warning "Neither nc nor telnet found, skipping network test"
        return 0
    fi
}

# Function to test database connection
test_database() {
    print_status "Testing database connection..."
    
    if ! command_exists psql; then
        print_error "psql command not found. Please install PostgreSQL client tools."
        return 1
    fi
    
    # Test connection
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT version();" >/dev/null 2>&1; then
        print_success "Database connection successful"
        return 0
    else
        print_error "Database connection failed"
        return 1
    fi
}

# Function to test SSL connection
test_ssl() {
    print_status "Testing SSL connection..."
    
    if [ "$DB_SSL_MODE" = "disable" ]; then
        print_warning "SSL is disabled (DB_SSL_MODE=disable)"
        return 0
    fi
    
    # Test SSL connection
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SHOW ssl;" >/dev/null 2>&1; then
        print_success "SSL connection successful"
        return 0
    else
        print_error "SSL connection failed"
        return 1
    fi
}

# Function to show connection details
show_connection_details() {
    print_status "Connection Details:"
    echo "  Host: $DB_HOST"
    echo "  Port: $DB_PORT"
    echo "  Database: $DB_NAME"
    echo "  User: $DB_USER"
    echo "  SSL Mode: $DB_SSL_MODE"
    echo "  Connect Timeout: $DB_CONNECT_TIMEOUT"
    echo "  Max Open Connections: $DB_MAX_OPEN_CONNS"
    echo "  Max Idle Connections: $DB_MAX_IDLE_CONNS"
    echo "  Connection Max Lifetime: $DB_CONN_MAX_LIFETIME"
}

# Function to show usage
show_usage() {
    echo "Monitor Agent Database Connection Test Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --help              Show this help message"
    echo "  --network           Test network connectivity only"
    echo "  --database          Test database connection only"
    echo "  --ssl               Test SSL connection only"
    echo "  --all               Run all tests"
    echo ""
    echo "Environment Variables:"
    echo "  DB_HOST             PostgreSQL host"
    echo "  DB_PORT             PostgreSQL port (default: 5432)"
    echo "  DB_NAME             Database name"
    echo "  DB_USER             Database user"
    echo "  DB_PASSWORD         Database password"
    echo "  DB_SSL_MODE         SSL mode"
    echo ""
    echo "Examples:"
    echo "  $0 --all                    # Run all tests"
    echo "  $0 --network                # Test network only"
    echo "  $0 --database               # Test database only"
}

# Main script logic
main() {
    # Load environment variables
    load_env
    
    # Check if any arguments provided
    if [ $# -eq 0 ]; then
        show_usage
        exit 1
    fi
    
    # Show connection details
    show_connection_details
    echo ""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help)
                show_usage
                exit 0
                ;;
            --network)
                test_network
                ;;
            --database)
                test_database
                ;;
            --ssl)
                test_ssl
                ;;
            --all)
                test_network && test_database && test_ssl
                print_success "All tests completed!"
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
        shift
    done
}

# Run main function with all arguments
main "$@"
