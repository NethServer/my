# Variables
BINARY_NAME=ssh-gateway
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Container runtime detection (Docker with Podman fallback)
DOCKER_CMD=$(shell which docker 2>/dev/null || echo "")
PODMAN_CMD=$(shell which podman 2>/dev/null || echo "")

# Choose container runtime
ifeq ($(DOCKER_CMD),)
    ifeq ($(PODMAN_CMD),)
        CONTAINER_CMD=echo "Error: Neither Docker nor Podman is installed" && exit 1
        CONTAINER_NAME=none
    else
        CONTAINER_CMD=$(PODMAN_CMD)
        CONTAINER_NAME=podman
    endif
else
    CONTAINER_CMD=$(DOCKER_CMD)
    CONTAINER_NAME=docker
endif

# Go build flags
LDFLAGS=-ldflags "-X github.com/nethesis/my/services/ssh-gateway/pkg/version.Version=$(VERSION) \
                  -X github.com/nethesis/my/services/ssh-gateway/pkg/version.Commit=$(COMMIT) \
                  -X github.com/nethesis/my/services/ssh-gateway/pkg/version.BuildTime=$(BUILD_TIME)"

# Default target
.PHONY: all
all: clean test build

# Run the application
.PHONY: run
run:
	@echo "Starting ssh-gateway development server..."
	@go run main.go

# Run the application with QA environment
.PHONY: run-qa
run-qa:
	@echo "Starting ssh-gateway QA server..."
	@ENV_FILE=.env.qa go run main.go

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) main.go
	@echo "Built $(BINARY_NAME) -> $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
.PHONY: build-all
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 main.go
	@echo "Built binaries for multiple platforms in $(BUILD_DIR)/"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out -coverpkg=./... ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linting
.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@gofmt -s -w .

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Install the binary
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) .

# Redis development commands
.PHONY: redis-up
redis-up:
	@echo "Starting shared Redis container with $(CONTAINER_NAME)..."
	@if $(CONTAINER_CMD) ps --format "{{.Names}}" | grep -q "^my-redis$$"; then \
		echo "Redis container already running"; \
		echo "REDIS_URL=redis://localhost:6379"; \
	else \
		$(CONTAINER_CMD) run -d --name my-redis -p 6379:6379 redis:7-alpine redis-server --save 60 1 --loglevel warning; \
		echo "REDIS_URL=redis://localhost:6379"; \
	fi

.PHONY: redis-down
redis-down:
	@echo "Stopping shared Redis container with $(CONTAINER_NAME)..."
	@$(CONTAINER_CMD) stop my-redis || true
	@$(CONTAINER_CMD) rm my-redis || true

.PHONY: redis-flush
redis-flush:
	@echo "Flushing Redis cache..."
	@$(CONTAINER_CMD) exec my-redis redis-cli FLUSHALL

.PHONY: redis-cli
redis-cli:
	@echo "Connecting to Redis CLI..."
	@$(CONTAINER_CMD) exec -it my-redis redis-cli

.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	@go mod download
	@if [ ! -f .env ]; then \
		echo "Creating .env from .env.example..."; \
		cp .env.example .env; \
	else \
		echo ".env already exists, skipping copy"; \
	fi
	@echo "Development environment ready!"

# Pre-commit checks
.PHONY: pre-commit
pre-commit: fmt lint test
	@echo "All pre-commit checks passed!"

# Show help
.PHONY: help
help:
	@echo "SSH Gateway Service - Container runtime: $(CONTAINER_NAME)"
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format code"
	@echo "  help          - Show this help"
	@echo "  install       - Install the binary"
	@echo "  lint          - Run linter"
	@echo "  pre-commit    - Run all pre-commit checks"
	@echo "  run           - Start development server"
	@echo "  run-qa        - Start QA server (uses .env.qa)"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  tidy          - Tidy dependencies"
	@echo ""
	@echo "Redis commands:"
	@echo "  redis-up      - Start Redis container"
	@echo "  redis-down    - Stop Redis container"
	@echo "  redis-flush   - Flush Redis cache"
	@echo "  redis-cli     - Connect to Redis CLI"
	@echo ""
	@echo "Development:"
	@echo "  dev-setup     - Setup development environment"
	@echo ""

# Default goal
.DEFAULT_GOAL := help
