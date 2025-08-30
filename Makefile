# Monitor Agent Makefile

# Variables
BINARY_NAME=monitor-agent
BUILD_DIR=build
DOCKER_IMAGE=monitor-agent
DOCKER_TAG=latest

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_UNIX=$(BUILD_DIR)/$(BINARY_NAME)

# Default target
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/monitor-agent

.PHONY: build-linux
build-linux: ## Build the application for Linux
	@echo "Building $(BINARY_NAME) for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) ./cmd/monitor-agent

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

.PHONY: test
test: ## Run unit tests
	@echo "Running unit tests..."
	$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration ./tests/integration/...

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	@echo "Running end-to-end tests..."
	$(GOTEST) -v -tags=e2e ./tests/e2e/...

.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...

.PHONY: mod-tidy
mod-tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	$(GOMOD) tidy

.PHONY: mod-download
mod-download: ## Download go modules
	@echo "Downloading go modules..."
	$(GOMOD) download

.PHONY: run
run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: run-scan
run-scan: ## Run a single scan
	@echo "Running scan..."
	./$(BUILD_DIR)/$(BINARY_NAME) scan

.PHONY: run-stats
run-stats: ## Show statistics
	@echo "Showing statistics..."
	./$(BUILD_DIR)/$(BINARY_NAME) stats

.PHONY: run-health
run-health: ## Run health check
	@echo "Running health check..."
	./$(BUILD_DIR)/$(BINARY_NAME) health

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f docker/Dockerfile .

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -it \
		--env-file .env \
		--network host \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-compose-up
docker-compose-up: ## Start services with Docker Compose
	@echo "Starting services with Docker Compose..."
	docker-compose -f docker/docker-compose.yml up -d

.PHONY: docker-compose-down
docker-compose-down: ## Stop services with Docker Compose
	@echo "Stopping services with Docker Compose..."
	docker-compose -f docker/docker-compose.yml down

.PHONY: docker-compose-logs
docker-compose-logs: ## Show Docker Compose logs
	@echo "Showing Docker Compose logs..."
	docker-compose -f docker/docker-compose.yml logs -f

.PHONY: install-deps
install-deps: ## Install development dependencies
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/DATA-DOG/go-sqlmock@latest

.PHONY: setup-db
setup-db: ## Setup database (requires remote PostgreSQL)
	@echo "Setting up database..."
	@if [ -z "$(DB_HOST)" ]; then \
		echo "Error: DB_HOST environment variable is required"; \
		exit 1; \
	fi
	./scripts/migrate.sh --all

.PHONY: migrate
migrate: ## Run database migrations
	@echo "Running database migrations..."
	./scripts/migrate.sh --migrate

.PHONY: migrate-validate
migrate-validate: ## Validate database connection
	@echo "Validating database connection..."
	./scripts/migrate.sh --validate

.PHONY: migrate-verify
migrate-verify: ## Verify migration results
	@echo "Verifying migration results..."
	./scripts/migrate.sh --verify

.PHONY: test-db-connection
test-db-connection: ## Test database connection
	@echo "Testing database connection..."
	./scripts/test-db-connection.sh --all

.PHONY: generate-mocks
generate-mocks: ## Generate mock files
	@echo "Generating mock files..."
	mockgen -source=internal/platforms/platform.go -destination=internal/platforms/mocks/platform_mock.go
	mockgen -source=internal/discovery/chaosdb/client.go -destination=internal/discovery/chaosdb/mocks/client_mock.go

.PHONY: security-scan
security-scan: ## Run security scan
	@echo "Running security scan..."
	gosec ./...

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

.PHONY: release
release: ## Create a release build
	@echo "Creating release build..."
	$(MAKE) clean
	$(MAKE) build-linux
	$(MAKE) docker-build

.PHONY: all
all: clean mod-download fmt vet lint test build ## Run all checks and build

.PHONY: ci
ci: mod-download fmt vet lint test test-coverage build ## Run CI pipeline
