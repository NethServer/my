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
3. **Service manifest** is exchanged — the system opens a control stream and sends the list of reachable services as JSON
4. **Diagnostics report** is sent — the system collects and pushes a health snapshot (CPU, RAM, disk, custom plugins)
5. **Operator requests** arrive as yamux streams with `CONNECT <service>\n` headers routing to the target service
6. **Reverse proxy** forwards HTTP/WebSocket traffic through the tunnel to remote services

The support service can also **push commands** to the tunnel-client by opening outbound yamux streams. The stream starts with a `COMMAND <version>\n` header followed by a JSON payload. Currently supported commands:

| Command | Description |
|:---|:---|
| `add_services` | Inject one or more static `host:port` services into the running session without reconnection |

### Session Lifecycle
- `pending` — Session created by backend, waiting for system to connect
- `active` — System connected, tunnel established
- `expired` — Session past `expires_at`, cleaned up by background cleaner
- `closed` — Session closed by operator or system disconnect

### Inter-Service Communication
- **Backend → Support**: Redis pub/sub on channel `support:commands` (`close` and `add_services` commands)
- **Backend → Support**: Internal HTTP endpoints with `X-Internal-Secret` header (proxy, terminal, services)
- **System → Support**: WebSocket with HTTP Basic Auth (tunnel establishment)
- **Support → System**: Outbound yamux COMMAND streams (server-initiated, e.g. `add_services`)

## Development

### Basic Commands
```bash
# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# Build support service
make build

# Build tunnel-client (linux/amd64)
make build-tunnel-client

# Build all binaries (support + tunnel-client)
make build-all

# Run server
make run

# Run tunnel-client locally
make run-tunnel-client

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
│       ├── main.go          # CLI entry point (flags, signal handling)
│       └── internal/
│           ├── config/      # ClientConfig, env parsing, helpers
│           ├── connection/  # WebSocket + yamux connection, reconnect loop
│           ├── diagnostics/ # Plugin runner, built-in system check, report aggregation
│           ├── discovery/   # Service discovery (Traefik, NethSecurity, static)
│           ├── models/      # ServiceInfo, ServiceManifest, ApiCliRoute
│           ├── stream/      # CONNECT protocol stream handler
│           └── terminal/    # PTY spawning, binary frame protocol
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

### Tunnel Client Configuration

All tunnel client settings are configured via environment variables or CLI flags.

Service exclusion patterns filter out services that are not useful for support operators:

```bash
# Via environment variable (comma-separated glob patterns)
EXCLUDE_PATTERNS="*-server-api,*-janus,*-middleware-*,*-provisioning,*-reports-api,*-cti-server-api,*-server-websocket,*-tancredi,*_loki,*_prometheus"

# Via CLI flag
tunnel-client --exclude "*-server-api,*-janus,*-middleware-*"
```

### Static Service Injection

Operators can add arbitrary `host:port` services to a running tunnel without restarting the tunnel-client. This is useful for services not auto-discovered via Traefik — for example the web management interface of a device on the customer's LAN (IP phone, managed switch, etc.).

**Flow:**

```
Operator clicks "Add service" in the UI
  → POST /api/support-sessions/:id/services  {name, target, label, tls}
  → Backend validates and publishes to Redis: {action: "add_services", session_id, services}
  → Support service opens an outbound yamux stream to the tunnel-client
  → Writes: COMMAND 1\n + JSON payload
  → Tunnel-client merges the new service into its local map and re-sends the manifest
  → Support service updates its service registry for that session
  → Operator can immediately open the new service via the proxy
```

**Example:** to access a Yealink phone's web UI at `192.168.1.100:443` on a customer's system, add a service with `target: 192.168.1.100:443` and `tls: true`. The phone's interface becomes available through the subdomain proxy as if the operator were on the same LAN.

Constraints: max 10 services per call, names must match `[a-zA-Z0-9][a-zA-Z0-9._-]*`, target must be `host:port`.

### Diagnostics Plugin System

At connect time, the tunnel-client collects a health report and pushes it to the support service. The report is stored with the session and shown to operators in the MY interface.

**Built-in plugin** (`system`): always runs, collects OS info, CPU load averages, RAM usage, disk usage, and uptime from `/proc` and `syscall.Statfs`.

**External plugins**: executable files dropped in `/usr/share/my/diagnostics.d/` (configurable via `DIAGNOSTICS_DIR`). Each plugin:
- Runs with a per-plugin timeout (default 10s, configurable via `DIAGNOSTICS_PLUGIN_TIMEOUT`)
- Writes JSON to stdout
- Uses exit code to signal status: `0` = ok, `1` = warning, `2` = critical

Plugin output format:
```json
{
  "id": "nethvoice",
  "name": "NethVoice",
  "status": "warning",
  "summary": "FreeSWITCH up, DB at 87% capacity",
  "checks": [
    { "name": "FreeSWITCH", "status": "ok", "value": "running" },
    { "name": "Asterisk CDR DB", "status": "warning", "value": "87% full" }
  ]
}
```

The overall session status is the worst status across all plugins. If the `id` or `name` fields are omitted, they are derived from the filename. If stdout is not valid JSON, the raw text is used as `summary`.

```bash
# Diagnostics flags
--diagnostics-dir string               # Default: /usr/share/my/diagnostics.d (env: DIAGNOSTICS_DIR)
--diagnostics-plugin-timeout duration  # Default: 10s (env: DIAGNOSTICS_PLUGIN_TIMEOUT)
--diagnostics-total-timeout duration   # Default: 30s (env: DIAGNOSTICS_TOTAL_TIMEOUT)
```

## Related
- [openapi.yaml](../../backend/openapi.yaml) - API specification
- [Backend](../../backend/README.md) - API server
- [Collect](../../collect/README.md) - Inventory collection service
- [Proxy](../../proxy/README.md) - Nginx reverse proxy
- [Project Overview](../../README.md) - Main documentation
