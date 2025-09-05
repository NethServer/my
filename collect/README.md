# Collect - System Inventory Collection Service

High-performance inventory collection service that handles thousands of systems reporting their inventory data every minute with real-time change detection and notifications.

## Quick Start

### Prerequisites
- Go 1.23+
- PostgreSQL 15+
- Redis 7+
- Docker/Podman

### Setup
```bash
# Setup development environment
make dev-setup

# Start PostgreSQL and Redis containers
make dev-up

# Start the application
make run

# Stop PostgreSQL and Redis when done
make dev-down
```

### Required Environment Variables
```bash
# Postgres URL
DATABASE_URL=postgresql://collect:collect@localhost:5432/noc

# Redis Configuration
REDIS_URL=redis://localhost:6379
REDIS_DB=1
REDIS_PASSWORD=
# Note: REDIS_PASSWORD can be empty for Redis without
```

### Optional Environment Variables
```bash
LISTEN_ADDRESS=127.0.0.1:8081
INVENTORY_MAX_AGE=90d
API_MAX_REQUEST_SIZE=10MB
LOG_LEVEL=info
```

## Architecture

### Inventory Processing Flow

**1. Data Collection**
- Systems POST inventory data to `/api/systems/inventory`
- HTTP Basic Auth (`system_id:system_secret`)
- Data queued in `collect:inventory`

**2. Inventory Processing**
- **Inventory Worker** processes batches, stores in `inventory_records`
- SHA-256 deduplication prevents duplicate processing
- Triggers diff computation for systems with previous records

**3. Change Detection**
- **Diff Worker** computes JSON diffs between current and previous inventory
- Categorizes changes (OS, hardware, network, features)
- Stores results in `inventory_diffs` with severity levels

**4. Notifications**
- **Notification Worker** processes significant changes
- Generates alerts based on configurable rules
- Stores notifications in `inventory_alerts`

**5. Retry & Cleanup**
- **Delayed Message Worker** handles failed jobs with exponential backoff
- **Cleanup Worker** removes old records (90-day retention)
- **Queue Monitor Worker** tracks system health and performance

### Queue Architecture
- `collect:inventory` → Raw inventory data
- `collect:processing` → Diff computation jobs
- `collect:notifications` → Alert notifications
- `{queue}:delayed` → Failed jobs with retry delays
- `{queue}:dead` → Jobs that exceeded max retry attempts

## Development

### Basic Commands
```bash
# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# Build
make build

# Run server
make run

# Run QA server (uses .env.qa)
make run-qa

# Test coverage
make test-coverage
```

### PostgreSQL Commands
```bash
# Start PostgreSQL container
make db-up

# Stop PostgreSQL container
make db-down

# Reset database
make db-reset

# Start full environment
make dev-up

# Stop full environment
make dev-down
```

### Redis Commands
```bash
# Start Redis container (Docker/Podman auto-detected)
make redis-up

# Stop Redis container
make redis-down

# Flush Redis cache
make redis-flush

# Connect to Redis CLI
make redis-cli

# Queue monitoring
redis-cli llen collect:inventory          # Pending inventory jobs
redis-cli llen collect:processing         # Pending diff jobs
redis-cli llen collect:notifications      # Pending notifications
redis-cli zcard collect:inventory:delayed # Delayed jobs (sorted set)
redis-cli llen collect:inventory:dead     # Dead letter queue

# View queue contents
redis-cli lrange collect:inventory 0 9
redis-cli zrange collect:inventory:delayed 0 9 WITHSCORES

# Clear queues (development only)
redis-cli del collect:inventory collect:processing collect:notifications
```

### Testing
```bash
# Submit inventory (requires system credentials from backend API)
curl -X POST http://localhost:8081/api/systems/inventory \
  -H "Content-Type: application/json" \
  -u "system_id:system_secret" \
  -d '{"system_id": "test", "timestamp": "2025-07-13T10:00:00Z", "data": {"os": {"name": "TestOS"}}}'
```

## Project Structure

```
collect/
├── main.go                 # Application entry point
├── configuration/          # Environment configuration
├── database/              # PostgreSQL connection and models
├── methods/               # HTTP request handlers
├── middleware/            # Authentication middleware
├── models/                # Data structures
├── queue/                 # Redis queue management
├── workers/               # Background processing workers
│   ├── inventory_worker.go        # Batch inventory processing
│   ├── diff_worker.go             # Change detection
│   ├── notification_worker.go     # Alert notifications
│   ├── cleanup_worker.go          # Data maintenance
│   ├── queue_monitor_worker.go    # Queue health monitoring
│   ├── delayed_message_worker.go  # Retry handling
│   └── manager.go                 # Worker orchestration
└── differ/                # JSON diff computation engine
```

## Related
- [openapi.yaml](../backend/openapi.yaml) - API specification
- [Backend](../backend/README.md) - API server
- [sync CLI](../sync/README.md) - RBAC configuration tool
- [Project Overview](../README.md) - Main documentation