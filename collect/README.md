# Collect - System Inventory Collection Service

**Collect** is a high-performance system inventory collection and processing service designed to handle thousands of systems reporting their inventory data every minute. It provides real-time change detection, monitoring, and alerting capabilities for system inventory management.

## 🚀 Features

### **High-Performance Inventory Collection**
- **Fast API Ingestion**: HTTP endpoint optimized for high-volume inventory data
- **Queue-Based Processing**: Redis-backed queue system for reliable data processing
- **Horizontal Scaling**: Multiple worker processes for concurrent data handling
- **Deduplication**: Automatic detection and skipping of duplicate inventory data

### **Intelligent Change Detection**
- **JSON Diff Engine**: Advanced diff computation for complex inventory data structures
- **Smart Filtering**: Filters out noise while preserving significant changes
- **Categorization**: Automatic categorization of changes (OS, hardware, network, features)
- **Severity Assessment**: Intelligent severity ranking (low, medium, high, critical)

### **Authentication & Security**
- **HTTP Basic Auth**: System credentials using `system_id:system_secret`
- **Credential Management**: Secure credential storage with SHA-256 hashing
- **Request Validation**: Comprehensive payload validation and size limits
- **Security Logging**: Automatic redaction of sensitive data in logs

### **Monitoring & Alerting**
- **Real-time Notifications**: Configurable notification system for inventory changes
- **Health Monitoring**: Comprehensive health checks for all system components
- **Statistics & Metrics**: Detailed processing statistics and performance metrics
- **Cleanup & Maintenance**: Automatic cleanup of old data and maintenance tasks

## 📋 API Endpoints

### **Inventory Collection**
```bash
POST /api/systems/inventory
Authorization: Basic <base64(system_id:system_secret)>
Content-Type: application/json

{
  "system_id": "17639",
  "timestamp": "2025-07-13T01:15:09.542Z",
  "data": {
    "os": {
      "name": "NethSec",
      "type": "nethsecurity",
      "family": "OpenWRT",
      "release": {
        "full": "8-24.10.0-ns.1.6.0-5-g0524860a0",
        "major": 7
      }
    },
    "networking": {
      "fqdn": "fw.nethesis.it"
    },
    "processors": {
      "count": "4",
      "models": ["Intel(R) Core(TM) i5-4570S CPU @ 2.90GHz"]
    },
    "memory": {
      "system": {
        "total_bytes": 7352455168,
        "used_bytes": 579198976,
        "available_bytes": 7352455168
      }
    }
  }
}
```

### **Health & Status**
- `GET /api/health` - Service health check
- `GET /api/stats` - Processing statistics (planned)
- `GET /api/format` - Expected inventory format documentation (planned)

## 🏗️ Architecture

```
Systems → POST /api/systems/inventory → Collect API → Redis Queue → Workers → PostgreSQL
                     ↓
            [Inventory Processor] → [Diff Processor] → [Notification Processor]
                     ↓                    ↓                      ↓
              [Database Storage]    [Change Detection]    [Alert System]
```

### **Core Components**

1. **HTTP API Server** (Gin-based)
   - Fast inventory ingestion endpoint
   - HTTP Basic authentication
   - Request validation and queuing

2. **Redis Queue System**
   - `collect:inventory` - Raw inventory data
   - `collect:processing` - Diff computation jobs
   - `collect:notifications` - Alert notifications
   - Delayed message handling for retries

3. **Background Workers**
   - **Inventory Processor**: Stores inventory data and detects duplicates
   - **Diff Processor**: Computes changes between inventory snapshots
   - **Notification Processor**: Sends alerts for significant changes
   - **Cleanup Worker**: Maintains database and removes old data
   - **Health Monitor**: Monitors system health and performance

4. **PostgreSQL Database**
   - `inventory_records` - Stored inventory snapshots
   - `inventory_diffs` - Computed changes between snapshots
   - `inventory_monitoring` - Monitoring rules and thresholds
   - `inventory_alerts` - Generated alerts and notifications
   - `system_credentials` - System authentication credentials

## 🛠️ Development Setup

### **Prerequisites**
- Go 1.23+
- PostgreSQL 15+
- Redis 7+
- Make (for build automation)
- Docker or Podman (for development containers)

### **Quick Start**

1. **Clone and Setup**
   ```bash
   cd collect
   make dev-setup
   ```

2. **Start Development Environment**
   ```bash
   # Start PostgreSQL and Redis containers
   make dev-env-up
   
   # In another terminal, start the application
   make run
   ```

3. **Test the API**
   ```bash
   # First, create system credentials (you'll need to do this via the backend API)
   # Then test inventory submission
   curl -X POST http://localhost:8081/api/systems/inventory \
     -H "Content-Type: application/json" \
     -u "system_id:system_secret" \
     -d '{
       "system_id": "test-system",
       "timestamp": "2025-07-13T10:00:00Z",
       "data": {"os": {"name": "TestOS"}}
     }'
   ```

### **Development Commands**

```bash
# Build and test
make build              # Build binary
make test              # Run tests
make test-coverage     # Run tests with coverage
make fmt               # Format code
make lint              # Run linter
make pre-commit        # Run all quality checks

# Database management
make db-up             # Start PostgreSQL container
make db-down           # Stop PostgreSQL container
make db-reset          # Reset database

# Redis management
make redis-up          # Start Redis container
make redis-down        # Stop Redis container
make redis-flush       # Clear Redis data

# Development environment
make dev-env-up        # Start full environment
make dev-env-down      # Stop full environment
```

## ⚙️ Configuration

### **Environment Variables**

| Variable | Default | Description |
|----------|---------|-------------|
| `LISTEN_ADDRESS` | `127.0.0.1:8081` | HTTP server bind address |
| `DATABASE_URL` | Required | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection URL |
| `REDIS_DB` | `1` | Redis database number |
| `WORKER_INVENTORY_COUNT` | `5` | Number of inventory worker processes |
| `WORKER_PROCESSING_COUNT` | `3` | Number of diff worker processes |
| `WORKER_NOTIFICATION_COUNT` | `2` | Number of notification worker processes |
| `INVENTORY_MAX_AGE` | `90d` | Maximum age for inventory data retention |
| `API_MAX_REQUEST_SIZE` | `10MB` | Maximum request payload size |
| `SYSTEM_SECRET_MIN_LENGTH` | `32` | Minimum length for system secrets |

### **Database Configuration**

The application automatically creates all necessary database tables on startup:

- **inventory_records**: Stores system inventory snapshots
- **inventory_diffs**: Records changes between inventory versions
- **inventory_monitoring**: Defines monitoring rules and thresholds
- **inventory_alerts**: Tracks generated alerts and notifications
- **system_credentials**: Manages system authentication

### **Queue Configuration**

Redis queues are automatically managed:

- **Main Queues**: `collect:inventory`, `collect:processing`, `collect:notifications`
- **Delayed Queues**: `{queue_name}:delayed` for retry handling
- **Dead Letter**: `{queue_name}:dead` for failed messages

## 🔍 Monitoring

### **Health Checks**

The service provides comprehensive health monitoring:

```bash
# Basic health check
curl http://localhost:8081/api/health

# Detailed system health (planned)
curl http://localhost:8081/api/health/detailed
```

### **Metrics**

Key metrics tracked:
- **Inventory Processing**: Records processed, processing time, queue lengths
- **Change Detection**: Diffs computed, significant changes detected
- **System Health**: Database connections, Redis connectivity, worker status
- **Error Rates**: Failed jobs, retry attempts, dead letter queue size

### **Logging**

Structured logging with automatic sensitive data redaction:

```bash
# View logs in development
make run 2>&1 | jq 'select(.level == "error")'

# Monitor specific components
make run 2>&1 | jq 'select(.component == "inventory-processor")'
```

## 🔒 Security

### **Authentication**
- **HTTP Basic Auth**: Each system uses `system_id:system_secret` credentials
- **Secret Hashing**: SHA-256 hashed secrets stored in database
- **Credential Caching**: Redis-based auth result caching with TTL

### **Data Protection**
- **Request Validation**: Comprehensive payload validation
- **Size Limits**: Configurable maximum request size (default 10MB)
- **Sensitive Data**: Automatic redaction in logs and error messages
- **Timestamp Validation**: Prevents replay attacks with timestamp windows

### **Access Control**
- **System Isolation**: Each system can only submit its own inventory data
- **Credential Management**: Secure credential creation and rotation support
- **Audit Logging**: Complete audit trail of all authentication events

## 🚀 Deployment

### **Production Environment**

1. **Database Setup**
   ```bash
   # Create PostgreSQL database
   createdb collect
   
   # The application will create tables automatically
   ```

2. **Configuration**
   ```bash
   # Copy and customize environment file
   cp .env.example .env
   # Edit .env with production values
   ```

3. **Build and Deploy**
   ```bash
   # Build for production
   make build
   
   # Deploy binary
   ./build/collect
   ```

### **Container Deployment**

```dockerfile
# Example Dockerfile (to be created)
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN make build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/build/collect .
CMD ["./collect"]
```

### **High Availability**

For production deployments:

- **Load Balancing**: Run multiple collect instances behind a load balancer
- **Database**: Use PostgreSQL with replication and backups
- **Redis**: Use Redis Cluster or Redis Sentinel for high availability
- **Monitoring**: Implement comprehensive monitoring and alerting

## 📊 Performance

### **Capacity Planning**

Tested performance characteristics:

- **Throughput**: 1000+ inventory submissions per minute per instance
- **Queue Processing**: 100+ diffs computed per minute
- **Storage**: Efficient JSON storage with optional compression
- **Memory**: ~100MB base memory usage, scales with queue size

### **Optimization Features**

- **Deduplication**: Automatic duplicate detection prevents unnecessary processing
- **Batch Processing**: Workers process multiple items efficiently
- **Connection Pooling**: Optimized database connection management
- **Caching**: Redis-based caching for frequently accessed data

## 🔧 Troubleshooting

### **Common Issues**

1. **Database Connection Errors**
   ```bash
   # Check database connectivity
   psql postgresql://collect:collect@localhost:5432/collect
   ```

2. **Redis Connection Issues**
   ```bash
   # Test Redis connectivity
   redis-cli -h localhost -p 6379 ping
   ```

3. **Queue Backlog**
   ```bash
   # Check queue lengths
   redis-cli llen collect:inventory
   redis-cli llen collect:processing
   ```

4. **Worker Health**
   ```bash
   # Check application logs for worker status
   make run 2>&1 | jq 'select(.component == "health-monitor")'
   ```

### **Debug Mode**

```bash
# Enable debug logging
LOG_LEVEL=debug make run

# Monitor specific system
make run 2>&1 | jq 'select(.system_id == "your-system-id")'
```

## 🤝 Integration

### **Backend Integration**

Collect integrates with the main backend API for:

- **System Management**: CRUD operations for systems via backend API
- **Credential Management**: System credential creation and management
- **User Access**: Inventory data access through backend RBAC system
- **Monitoring**: Shared monitoring and alerting infrastructure

### **System Integration**

Systems should integrate by:

1. **Registration**: Create system via backend API to get credentials
2. **Data Collection**: Gather system inventory in expected JSON format
3. **Submission**: POST inventory data to collect API with authentication
4. **Monitoring**: Monitor API responses for errors and retry as needed

---

**Collect** - High-performance system inventory collection and change detection service, part of the Nethesis Operation Center ecosystem.