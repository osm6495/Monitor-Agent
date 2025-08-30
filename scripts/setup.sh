#!/bin/bash

# Monitor Agent Setup Script
# This script helps set up the Monitor Agent for development or production

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

# Function to check Go version
check_go_version() {
    if ! command_exists go; then
        print_error "Go is not installed. Please install Go 1.21 or later."
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    REQUIRED_VERSION="1.21"
    
    if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
        print_error "Go version $GO_VERSION is too old. Please install Go 1.21 or later."
        exit 1
    fi
    
    print_success "Go version $GO_VERSION is compatible"
}

# Function to check Docker
check_docker() {
    if ! command_exists docker; then
        print_warning "Docker is not installed. Docker is required for containerized deployment."
        return 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        print_warning "Docker is not running. Please start Docker."
        return 1
    fi
    
    print_success "Docker is available"
    return 0
}

# Function to setup environment file
setup_env() {
    if [ ! -f .env ]; then
        print_status "Creating .env file from template..."
        cp env.example .env
        print_success "Created .env file. Please edit it with your API keys and database configuration."
    else
        print_warning ".env file already exists. Skipping creation."
    fi
}

# Function to install dependencies
install_dependencies() {
    print_status "Installing Go dependencies..."
    go mod download
    
    print_status "Installing development tools..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    go install github.com/DATA-DOG/go-sqlmock@latest
    
    print_success "Dependencies installed"
}

# Function to setup database
setup_database() {
    if [ "$1" = "docker" ]; then
        print_warning "Docker database setup is deprecated. Please use remote PostgreSQL."
        print_status "Please configure your remote PostgreSQL connection in .env file"
        print_status "Example configuration:"
        echo "DB_HOST=your-remote-postgres-host.com"
        echo "DB_PORT=5432"
        echo "DB_NAME=monitor_agent"
        echo "DB_USER=monitor_agent"
        echo "DB_PASSWORD=your_secure_password"
        echo "DB_SSL_MODE=require"
    else
        print_status "Please configure your remote PostgreSQL connection in .env file"
        print_status "Required environment variables:"
        echo "- DB_HOST: Your remote PostgreSQL host"
        echo "- DB_PORT: PostgreSQL port (default: 5432)"
        echo "- DB_NAME: Database name"
        echo "- DB_USER: Database user"
        echo "- DB_PASSWORD: Database password"
        echo "- DB_SSL_MODE: SSL mode (require, verify-full, etc.)"
    fi
}

# Function to run database migrations
run_migrations() {
    print_status "Running database migrations..."
    
    # Check if database is accessible
    if ! command_exists psql; then
        print_warning "psql not found. Please install PostgreSQL client tools."
        return 1
    fi
    
    # Try to run migration
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f internal/database/migrations/001_initial_schema.sql; then
        print_success "Database migrations completed"
    else
        print_error "Failed to run database migrations. Please check your database connection."
        return 1
    fi
}

# Function to build the application
build_application() {
    print_status "Building Monitor Agent..."
    
    if make build; then
        print_success "Application built successfully"
    else
        print_error "Failed to build application"
        exit 1
    fi
}

# Function to run tests
run_tests() {
    print_status "Running tests..."
    
    if make test; then
        print_success "Tests passed"
    else
        print_error "Tests failed"
        exit 1
    fi
}

# Function to show usage
show_usage() {
    echo "Monitor Agent Setup Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --help              Show this help message"
    echo "  --env               Setup environment file"
    echo "  --deps              Install dependencies"
    echo "  --db-docker         Setup database with Docker"
    echo "  --db-local          Setup database locally (manual)"
    echo "  --migrate           Run database migrations"
    echo "  --build             Build the application"
    echo "  --test              Run tests"
    echo "  --all               Run all setup steps"
    echo ""
    echo "Examples:"
    echo "  $0 --all                    # Complete setup"
    echo "  $0 --env --deps --build     # Setup for development"
    echo "  $0 --db-docker --migrate    # Setup database only"
}

# Main script logic
main() {
    print_status "Starting Monitor Agent setup..."
    
    # Check prerequisites
    check_go_version
    check_docker
    
    # Parse command line arguments
    if [ $# -eq 0 ]; then
        show_usage
        exit 1
    fi
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help)
                show_usage
                exit 0
                ;;
            --env)
                setup_env
                ;;
            --deps)
                install_dependencies
                ;;
            --db-docker)
                setup_database docker
                ;;
            --db-local)
                setup_database local
                ;;
            --migrate)
                run_migrations
                ;;
            --build)
                build_application
                ;;
            --test)
                run_tests
                ;;
            --all)
                setup_env
                install_dependencies
                setup_database docker
                run_migrations
                build_application
                run_tests
                print_success "Setup completed successfully!"
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
