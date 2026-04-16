# Collect - System Inventory Collection Service

High-performance inventory collection service that handles thousands of systems reporting their inventory data every minute with real-time change detection and notifications.

## Quick Start

### Prerequisites
- Go 1.24+
- PostgreSQL 15+
- Redis 7+
- Docker/Podman

### Setup

> **Note:** Collect shares the same PostgreSQL and Redis containers with the backend.
> If you already started them with `cd backend && make dev-up`, you can skip `make dev-up` here.

```bash
# Setup development environment
make dev-setup

# Start PostgreSQL and Redis containers (skip if already running from backend)
make dev-up

# Start the application (port 8081)
make run

# Stop PostgreSQL and Redis when done
make dev-down
```

### Required Environment Variables
```bash
# Postgres URL
DATABASE_URL=postgresql://noc_user:noc_password@localhost:5432/noc?sslmode=disable

# Redis Configuration
REDIS_URL=redis://localhost:6379
REDIS_DB=1
REDIS_PASSWORD=
# Note: REDIS_PASSWORD can be empty for Redis without authentication
```

### Optional Environment Variables
```bash
LISTEN_ADDRESS=127.0.0.1:8081
API_MAX_REQUEST_SIZE=10MB
HEARTBEAT_TIMEOUT_MINUTES=10
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
- **Cleanup Worker** applies exponential retention to `inventory_records` (see below); `inventory_diffs` are never deleted
- **Queue Monitor Worker** tracks system health and performance

**6. Heartbeat Monitoring**
- **Heartbeat Monitor Cron** runs every 60 seconds
- Automatically updates system status based on heartbeat freshness:
  - `unknown` → `active` when first heartbeat arrives
  - `inactive` → `active` when fresh heartbeat arrives (< 10 minutes)
  - `active` → `inactive` when heartbeat is stale (> 10 minutes)
- Configurable timeout via `HEARTBEAT_TIMEOUT_MINUTES` (default: 10 minutes)

### Queue Architecture
- `collect:inventory` → Raw inventory data
- `collect:processing` → Diff computation jobs
- `collect:notifications` → Alert notifications
- `{queue}:delayed` → Failed jobs with retry delays
- `{queue}:dead` → Jobs that exceeded max retry attempts

### Data Retention Policy

**`inventory_diffs`** are never deleted. They are the source of truth for the timeline feature (`/inventory/timeline`) and are self-contained (each diff stores `field_path`, `previous_value`, `current_value`).

**`inventory_records`** (full JSON snapshots) use exponential retention — more frequent near the present, progressively sparser further back:

| Age | Retention |
|-----|-----------|
| Last 7 days | All records preserved |
| 7 days – 1 month | 1 per day |
| 1 month – 3 months | 1 per week |
| 3 months – 1 year | 1 per month |
| Older than 1 year | 1 per quarter |

The **first** record per system (inventory baseline) and the **latest** record (current state, used by `/inventory/latest`) are always preserved regardless of age.

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
# Submit heartbeat (requires system credentials from backend API)
# Authentication via HTTP Basic Auth: system_key:system_secret
curl -X POST http://localhost:8081/api/systems/heartbeat \
  -u "NOC-XXXX-XXXX-XXXX:your_system_secret"

# Submit inventory (requires system credentials from backend API)
curl -X POST http://localhost:8081/api/systems/inventory \
  -H "Content-Type: application/json" \
  -u "system_key:system_secret" \
  -d '{"system_id": "test", "timestamp": "2025-07-13T10:00:00Z", "data": {"os": {"name": "TestOS"}}}'
```

## Project Structure

```
collect/
├── main.go                 # Application entry point
├── configuration/          # Environment configuration
├── cron/                  # Scheduled jobs
│   └── heartbeat_monitor.go       # System status monitoring
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

## Machine-scoped Alertmanager access

The Mimir Alertmanager endpoints (`/services/mimir/alertmanager/api/v2/*`) implement strict per-machine scoping to ensure systems can only access their own alerts and silences.

### How scoping works

Each system (identified by `system_key`) is automatically restricted to see and manage only its own data:

- **GET /alerts**: The proxy injects a `system_key` filter into the query, limiting results to this system's alerts
- **POST /alerts**: The `system_key`, `system_id`, and `organization_id` labels are injected and override any client values
- **GET /silences**: The proxy injects a `system_key` filter to scope results to this system's silences
- **POST /silences**: The proxy injects a `system_key` matcher, overwriting any client-supplied `system_key` matcher. This ensures the silence can only target this system's alerts
- **GET /silences/{id}**: The proxy fetches the silence from Mimir and verifies it contains an exact `system_key` matcher matching this system. Denies access (403) if the silence does not belong to this system
- **DELETE /silences/{id}**: Same ownership verification as GET, then deletes only if verified

### Security properties

- Systems cannot view, create, or modify silences for other systems
- Silence matchers cannot be bypassed — a `system_key` matcher is always enforced server-side
- The `system_key` label in alerts is always server-sourced (injected via `injectLabels` in `mimir.go`), never trusted from client input
- Failed ownership checks are logged with system and path details for audit purposes

## Alert annotation templating

Alert annotations support Go text/template syntax. When alerts are posted to `/api/services/mimir/alertmanager/api/v2/alerts`, the proxy processes any template expressions in annotation values, substituting alert labels.

### Template syntax

Use standard Go template syntax with alert labels as data:

```json
{
  "labels": {
    "severity": "critical",
    "alertname": "DiskFull",
    "system_key": "SYS-001"
  },
  "annotations": {
    "summary": "Alert {{.alertname}} has severity {{.severity}}",
    "description": "System: {{.system_key}}"
  }
}
```

Result after templating:
```json
{
  "annotations": {
    "summary": "Alert DiskFull has severity critical",
    "description": "System: SYS-001"
  }
}
```

### Behavior

- Only annotations containing `{{` are processed
- Labels are passed as template data (accessible via `.fieldname`)
- Non-existent labels render as `<no value>`
- Invalid template syntax is logged as a warning; the annotation remains unchanged
- Non-string annotation values are preserved as-is
- Static annotations (without template syntax) pass through unchanged