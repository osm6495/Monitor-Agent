#!/bin/bash

# Monitor Agent Database Migration Script
# This script helps set up the database schema on a remote PostgreSQL instance

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

# Function to validate database connection
validate_connection() {
    print_status "Validating database connection..."
    
    if ! command_exists psql; then
        print_error "psql command not found. Please install PostgreSQL client tools."
        exit 1
    fi
    
    # Test connection
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" >/dev/null 2>&1; then
        print_success "Database connection successful"
    else
        print_error "Failed to connect to database. Please check your configuration."
        print_error "Required environment variables:"
        echo "  - DB_HOST: $DB_HOST"
        echo "  - DB_PORT: $DB_PORT"
        echo "  - DB_NAME: $DB_NAME"
        echo "  - DB_USER: $DB_USER"
        echo "  - DB_PASSWORD: [hidden]"
        echo "  - DB_SSL_MODE: $DB_SSL_MODE"
        exit 1
    fi
}

# Function to run migrations
run_migrations() {
    print_status "Running database migrations..."
    
    # Check if migration file exists
    MIGRATION_FILE="internal/database/migrations/001_initial_schema.sql"
    if [ ! -f "$MIGRATION_FILE" ]; then
        print_error "Migration file not found: $MIGRATION_FILE"
        exit 1
    fi
    
    # Run migration
    if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$MIGRATION_FILE"; then
        print_success "Database migrations completed successfully"
    else
        print_error "Failed to run database migrations"
        exit 1
    fi
}

# Function to verify migration
verify_migration() {
    print_status "Verifying migration..."
    
    # Check if tables exist
    TABLES=("platforms" "programs" "assets" "asset_responses" "scans")
    
    for table in "${TABLES[@]}"; do
        if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\dt $table" >/dev/null 2>&1; then
            print_success "Table '$table' exists"
        else
            print_error "Table '$table' not found"
            exit 1
        fi
    done
    
    print_success "All tables verified successfully"
}

# Function to show usage
show_usage() {
    echo "Monitor Agent Database Migration Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --help              Show this help message"
    echo "  --validate          Validate database connection only"
    echo "  --migrate           Run database migrations"
    echo "  --verify            Verify migration results"
    echo "  --all               Run all steps (validate, migrate, verify)"
    echo ""
    echo "Environment Variables:"
    echo "  DB_HOST             PostgreSQL host"
    echo "  DB_PORT             PostgreSQL port (default: 5432)"
    echo "  DB_NAME             Database name"
    echo "  DB_USER             Database user"
    echo "  DB_PASSWORD         Database password"
    echo "  DB_SSL_MODE         SSL mode (require, verify-full, etc.)"
    echo ""
    echo "Examples:"
    echo "  $0 --all                    # Run complete migration"
    echo "  $0 --validate               # Test connection only"
    echo "  $0 --migrate                # Run migrations only"
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
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help)
                show_usage
                exit 0
                ;;
            --validate)
                validate_connection
                ;;
            --migrate)
                validate_connection
                run_migrations
                ;;
            --verify)
                verify_migration
                ;;
            --all)
                validate_connection
                run_migrations
                verify_migration
                print_success "Database migration completed successfully!"
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
