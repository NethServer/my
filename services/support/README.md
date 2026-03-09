# Support - Remote Support Session Service

WebSocket tunnel-based remote support service that enables operators to access remote systems through multiplexed yamux sessions.

## Quick Start

### Prerequisites
- Go 1.24+
- PostgreSQL 15+
- Redis 7+
- Docker/Podman

### Setup

> **Note:** Support shares the same PostgreSQL and Redis containers with the backend.
> If you already started them with `cd backend && make dev-up`, you can skip `make dev-up` here.

```bash
# Setup development environment
make dev-setup

# Start PostgreSQL and Redis containers (skip if already running from backend)
make dev-up

# Start the application (port 8082)
make run

# Stop PostgreSQL and Redis when done
make dev-down
```

### Required Environment Variables
```bash
# Database
DATABASE_URL=postgresql://noc_user:noc_password@localhost:5432/noc?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379
REDIS_DB=2

# Internal authentication (shared secret with backend)
INTERNAL_SECRET=change-me-to-a-random-secret-min-32-chars
```

### Optional Environment Variables
```bash
LISTEN_ADDRESS=127.0.0.1:8082
LOG_LEVEL=info
LOG_FORMAT=console
SYSTEM_AUTH_CACHE_TTL=24h
SYSTEM_SECRET_MIN_LENGTH=32
SESSION_DEFAULT_DURATION=24h
SESSION_CLEANER_INTERVAL=5m
TUNNEL_GRACE_PERIOD=2m
MAX_TUNNELS=1000
MAX_SESSIONS_PER_SYSTEM=5
```

## Architecture

### Tunnel Flow

1. **System connects** via WebSocket with HTTP Basic Auth (same credentials as collect)
2. **yamux session** multiplexes streams over a single WebSocket connection
3. **Service manifest** is exchanged — the system advertises available services (e.g., cluster-admin, SSH)
4. **Operator requests** arrive as yamux streams with CONNECT headers routing to the target service
5. **Reverse proxy** forwards HTTP/WebSocket traffic through the tunnel to remote services

### Session Lifecycle
- `pending` — Session created by backend, waiting for system to connect
- `active` — System connected, tunnel established
- `expired` — Session past `expires_at`, cleaned up by background cleaner
- `closed` — Session closed by operator or system disconnect

### Inter-Service Communication
- **Backend → Support**: Redis pub/sub on channel `support:commands` (close sessions)
- **Backend → Support**: Internal HTTP endpoints with `X-Internal-Secret` header (proxy, terminal, services)
- **System → Support**: WebSocket with HTTP Basic Auth (tunnel establishment)

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
```

### Redis Commands
```bash
# Start Redis container
make redis-up

# Stop Redis container
make redis-down

# Flush Redis cache
make redis-flush

# Connect to Redis CLI
make redis-cli
```

## Project Structure

```
services/support/
├── main.go                  # Server entry point
├── cmd/
│   └── tunnel-client/       # Client binary deployed on remote systems
├── configuration/           # Environment configuration
├── database/                # PostgreSQL connection
├── helpers/                 # SHA256 verification
├── logger/                  # Structured logging (zerolog)
├── methods/                 # HTTP/WebSocket handlers
│   ├── tunnel.go            # WebSocket tunnel endpoint
│   ├── proxy.go             # HTTP reverse proxy through tunnel
│   ├── terminal.go          # Web terminal (WebSocket-to-SSH)
│   └── commands.go          # Redis pub/sub command listener
├── middleware/               # Auth and rate limiting
│   ├── auth.go              # HTTP Basic Auth (SHA256) + caching
│   └── ratelimit.go         # Tunnel connection rate limiting
├── models/                  # Data structures
├── queue/                   # Redis client
├── response/                # HTTP response helpers
├── session/                 # Session CRUD and background cleaner
├── tunnel/                  # yamux tunnel manager and protocol
│   ├── manager.go           # In-memory tunnel registry
│   ├── protocol.go          # CONNECT header protocol
│   └── stream.go            # WebSocket-to-net.Conn adapter
├── pkg/version/             # Build version info
└── .env.example             # Environment variables template
```

## Related
- [openapi.yaml](../../backend/openapi.yaml) - API specification
- [Backend](../../backend/README.md) - API server
- [Collect](../../collect/README.md) - Inventory collection service
- [Proxy](../../proxy/README.md) - Nginx reverse proxy
- [Project Overview](../../README.md) - Main documentation
