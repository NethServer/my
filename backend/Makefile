# Variables
BINARY_NAME=backend
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
LDFLAGS=-ldflags "-X main.Version=$(VERSION) \
                  -X main.Commit=$(COMMIT) \
                  -X main.BuildTime=$(BUILD_TIME)"

# Default target
.PHONY: all
all: clean test build

# Run the application
.PHONY: run
run:
	@echo "Starting development server..."
	@go run main.go

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
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 main.go
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 main.go
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 main.go
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe main.go
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

# Development setup
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

# Redis development commands
.PHONY: redis-up
redis-up:
	@echo "Starting Redis container with $(CONTAINER_NAME)..."
	@$(CONTAINER_CMD) run -d --name backend-redis -p 6379:6379 redis:7-alpine redis-server --save 60 1 --loglevel warning

.PHONY: redis-down
redis-down:
	@echo "Stopping Redis container with $(CONTAINER_NAME)..."
	@$(CONTAINER_CMD) stop backend-redis || true
	@$(CONTAINER_CMD) rm backend-redis || true

.PHONY: redis-flush
redis-flush:
	@echo "Flushing Redis cache..."
	@$(CONTAINER_CMD) exec backend-redis redis-cli FLUSHALL

.PHONY: redis-cli
redis-cli:
	@echo "Connecting to Redis CLI..."
	@$(CONTAINER_CMD) exec -it backend-redis redis-cli

# Show container runtime status
.PHONY: container-status
container-status:
	@echo "Container runtime detection:"
	@echo "  Docker: $(if $(DOCKER_CMD),✓ $(DOCKER_CMD),✗ Not found)"
	@echo "  Podman: $(if $(PODMAN_CMD),✓ $(PODMAN_CMD),✗ Not found)"
	@echo ""
	@echo "Selected runtime: $(CONTAINER_NAME)"

# Show help
.PHONY: help
help:
	@echo "Backend API - Container runtime: $(CONTAINER_NAME)"
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  clean         - Clean build artifacts"
	@echo "  dev-setup     - Setup development environment"
	@echo "  fmt           - Format code"
	@echo "  help          - Show this help"
	@echo "  install       - Install the binary"
	@echo "  lint          - Run linter"
	@echo "  run           - Start development server"
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

# Default goal
.DEFAULT_GOAL := help