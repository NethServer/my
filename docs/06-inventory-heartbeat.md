# Inventory and Heartbeat

Learn how external systems send inventory data and heartbeat signals to My platform.

## Overview

After [system registration](05-system-registration.md), external systems communicate with My through two mechanisms:

1. **Inventory**: Complete system information snapshot (hardware, software, configuration)
2. **Heartbeat**: Periodic "I'm alive" signal to indicate system is online

Both operations use **HTTP Basic Authentication** with the registered credentials.

## Authentication

### Credentials

Use the credentials obtained during system lifecycle:

- **Username**: `system_key` (received at registration)
- **Password**: `system_secret` (from system creation)

### HTTP Basic Auth

**Header format:**
```
Authorization: Basic base64(system_key:system_secret)
```

**Example:**
```
system_key: NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE
system_secret: my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0

Base64 encode: "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE:my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

Authorization: Basic bXlfc3lzX2FiYzEyM2RlZjQ1NjpteV9hMWIyYzNkNGU1ZjZnN2g4aTlqMC5rMWwybTNuNG81cDZxN3I4czl0MHUxdjJ3M3g0eTV6NmE3YjhjOWQw
```

**Most HTTP libraries handle this automatically:**
```python
import requests

requests.post(url,
    auth=('NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE', 'my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0'),
    json=data
)
```

## Heartbeat

Heartbeat is a simple signal to indicate the system is alive and reachable.

### Purpose

- Detect when systems go offline
- Trigger alerts for dead systems
- Monitor system reliability

### Endpoint

```
POST https://my.nethesis.it/collect/api/systems/heartbeat
```

**Note:** Collect service runs on **/collect** (different from main backend on **/backend/**)

### Request

**Headers:**
```
Authorization: Basic <credentials>
Content-Type: application/json
```

**Body:**
```json
{}
```

**Empty JSON object** - no data needed!

### Response

**Success (HTTP 200):**
```json
{
  "code": 200,
  "message": "heartbeat acknowledged",
  "data": {
    "system_key": "NOC-80F8-89A4-40B0-4AE9-A670-7C5F-99B3-F3EA",
    "acknowledged": true,
    "last_heartbeat": "2025-11-07T10:37:27.360343+01:00"
  }
}
```

### Frequency

**Recommended:** Every 5 minutes

**Why 5 minutes?**
- Platform considers system "active" if heartbeat < 15 minutes
- 5-minute interval provides 3 missed beats before marking dead
- Balance between network traffic and responsiveness

### Example Implementation

**Python:**
```python
import requests
import time

COLLECT_URL = "https://my.nethesis.it/collect"
SYSTEM_KEY = "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE"
SYSTEM_SECRET = "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

def send_heartbeat():
    """Send heartbeat to My platform"""
    try:
        response = requests.post(
            f"{COLLECT_URL}/api/systems/heartbeat",
            auth=(SYSTEM_KEY, SYSTEM_SECRET),
            json={},
            timeout=10
        )
        response.raise_for_status()
        print("Heartbeat sent successfully")
        return True
    except Exception as e:
        print(f"Heartbeat failed: {e}")
        return False

# Send heartbeat every 5 minutes
while True:
    send_heartbeat()
    time.sleep(300)  # 5 minutes
```

**Bash (cron):**
```bash
#!/bin/bash
# /usr/local/bin/my-heartbeat.sh

COLLECT_URL="https://my.nethesis.it/collect"
SYSTEM_KEY="NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE"
SYSTEM_SECRET="my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

curl -s -X POST "$COLLECT_URL/api/systems/heartbeat" \
  -u "$SYSTEM_KEY:$SYSTEM_SECRET" \
  -H "Content-Type: application/json" \
  -d '{}' > /dev/null
```

**Crontab entry (every 5 minutes):**
```
*/5 * * * * /usr/local/bin/my-heartbeat.sh
```

### Heartbeat Status

Systems are classified based on heartbeat:

| Status | Condition | Color | Meaning |
|--------|-----------|-------|---------|
| **Active** | < 15 minutes | 🟢 Green | System is healthy |
| **Inactive** | ≥ 15 minutes | 🟡 Yellow | System is offline |
| **Unknown** | Never sent | ⚪ Gray | Never communicated |

## Inventory

Inventory is a complete snapshot of system configuration and installed software.

### Purpose

- Track system hardware and software
- Detect configuration changes
- Monitor software versions
- Audit system compliance
- Track inventory history

### Endpoint

```
POST https://my.nethesis.it/collect/api/systems/inventory
```

**Note:** Same collect service, port **8081**

### Request

**Headers:**
```
Authorization: Basic <credentials>
Content-Type: application/json
```

**Body Structure:**
```json
{
  "fqdn": "server.example.com",
  "ipv4_address": "192.168.1.100",
  "ipv6_address": "2001:db8::1",
  "version": "8.0.1",
  "os": {
    "name": "Rocky Linux",
    "version": "9.3",
    "kernel": "5.14.0-362.8.1.el9_3.x86_64"
  },
  "hardware": {
    "cpu_model": "Intel Xeon Gold 6248R",
    "cpu_cores": 8,
    "cpu_threads": 16,
    "memory_total_gb": 32,
    "disk_total_gb": 500
  },
  "network": {
    "hostname": "server01",
    "interfaces": {
      "eth0": {
        "ip": "192.168.1.100",
        "netmask": "255.255.255.0",
        "mac": "00:1a:2b:3c:4d:5e"
      },
      "eth1": {
        "ip": "10.0.0.10",
        "netmask": "255.255.0.0",
        "mac": "00:1a:2b:3c:4d:5f"
      }
    }
  },
  "services": {
    "nginx": {
      "version": "1.24.0",
      "status": "running"
    },
    "postgresql": {
      "version": "15.5",
      "status": "running"
    },
    "redis": {
      "version": "7.2.3",
      "status": "running"
    }
  },
  "features": {
    "docker": {
      "enabled": true,
      "version": "24.0.7"
    },
    "firewall": {
      "enabled": true,
      "type": "nftables"
    }
  },
  "custom": {
    "environment": "production",
    "datacenter": "EU-West-1",
    "backup_enabled": true
  }
}
```

### Response

**Success (HTTP 200):**
```json
{
  "code": 200,
  "message": "Inventory received and queued for processing",
  "data": {
    "data_size": 16433,
    "message": "Your inventory data has been received and will be processed shortly",
    "queue_status": "queued",
    "system_id": "0a98637c-077b-428a-8e57-c2fbb892051a",
    "timestamp": "2025-11-07T10:39:05.897352+01:00"
  }
}
```

### Frequency

**Recommended:** Every 6 hours (4 times per day)

**Why 6 hours?**
- Balance between freshness and network/storage load
- Captures daily changes
- Reduces database growth
- Sufficient for most monitoring needs

**Special cases:**
- **After system changes**: Send immediately
- **During updates**: Send before and after
- **On-demand**: Admin can trigger via API

### Inventory Schema

#### Required Fields

**Minimum required data:**
```json
{
  "fqdn": "server.example.com",
  "ipv4_address": "192.168.1.100",
  "os": {
    "name": "NethSec",
    "type": "nethsecurity",
    "family": "OpenWRT",
    "release": {
        "full": "8.6.0-dev+43d54cd33.20251020175318",
        "major": 7
    }
  }
}
```

#### Optional Sections

All other sections are optional but recommended:

- `os`: Operating system information
- `hardware`: Physical/virtual hardware specs
- `network`: Network configuration
- `services`: Installed services and versions
- `features`: Enabled features and capabilities
- `custom`: Any custom data (free-form)

#### Auto-Detection

Some fields are auto-detected by platform:

- **System Type**: Detected from inventory data (ns8, nsec, etc.)
- **Status**: Auto-updated based on heartbeat
- **Last Update**: Timestamp of inventory receipt

### Example Implementation

**Python:**
```python
import requests
import platform
import psutil
import json

COLLECT_URL = "https://my.nethesis.it/collect"
SYSTEM_KEY = "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE"
SYSTEM_SECRET = "my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

def collect_inventory():
    """Collect system inventory"""
    return {
        "fqdn": platform.node(),
        "ipv4_address": get_primary_ip(),  # Your implementation
        "version": "8.0.1",
        "os": {
            "name": platform.system(),
            "version": platform.release(),
            "kernel": platform.version()
        },
        "hardware": {
            "cpu_cores": psutil.cpu_count(logical=False),
            "cpu_threads": psutil.cpu_count(logical=True),
            "memory_total_gb": round(psutil.virtual_memory().total / (1024**3), 2)
        },
        "services": collect_services(),  # Your implementation
        "features": collect_features(),  # Your implementation
    }

def send_inventory():
    """Send inventory to My platform"""
    try:
        inventory = collect_inventory()

        response = requests.post(
            f"{COLLECT_URL}/api/systems/inventory",
            auth=(SYSTEM_KEY, SYSTEM_SECRET),
            json=inventory,
            timeout=30
        )
        response.raise_for_status()

        data = response.json()
        print(f"Inventory sent successfully")
        print(f"Changes detected: {data['data']['changes_detected']}")
        return True

    except Exception as e:
        print(f"Inventory send failed: {e}")
        return False

# Send inventory every 6 hours
import time
while True:
    send_inventory()
    time.sleep(21600)  # 6 hours
```

**Bash:**
```bash
#!/bin/bash
# /usr/local/bin/my-inventory.sh

COLLECT_URL="https://my.nethesis.it/collect"
SYSTEM_KEY="NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE"
SYSTEM_SECRET="my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"

# Collect inventory (example - customize for your system)
INVENTORY=$(cat <<EOF
{
  "fqdn": "$(hostname -f)",
  "ipv4_address": "$(hostname -I | awk '{print $1}')",
  "version": "8.0.1",
  "os": {
    "name": "$(uname -s)",
    "version": "$(uname -r)"
  },
  "hardware": {
    "cpu_cores": $(nproc),
    "memory_total_gb": $(free -g | awk '/^Mem:/{print $2}')
  }
}
EOF
)

# Send to platform
curl -X POST "$COLLECT_URL/api/systems/inventory" \
  -u "$SYSTEM_KEY:$SYSTEM_SECRET" \
  -H "Content-Type: application/json" \
  -d "$INVENTORY"
```

## Change Detection

My automatically detects changes between inventory snapshots.

### What is Tracked

- Hardware changes (CPU, memory, disk)
- Software version changes
- Service additions/removals
- Network configuration changes
- Feature toggles
- Custom field modifications

### Change Categories

Changes are categorized by type:

- **OS**: Operating system and kernel
- **Hardware**: Physical/virtual hardware
- **Network**: Network interfaces and configuration
- **Features**: Enabled features and capabilities
- **Services**: Installed services
- **System**: General system settings

### Change Severity

Each change has a severity level:

- **Critical**: Requires immediate attention (e.g., hardware failure)
- **High**: Important changes (e.g., OS upgrade)
- **Medium**: Notable changes (e.g., service update)
- **Low**: Minor changes (e.g., metrics update)

### Viewing Changes

**In Admin Panel:**
1. Navigate to **Systems** > **System Details**
2. Click **Inventory** tab
3. View **Changes** section
4. See detailed diff between versions

**Change types:**
- 🟢 **Create**: New field added
- 🟡 **Update**: Field value changed
- 🔴 **Delete**: Field removed

**Example change log:**
```
[2025-11-06 10:30] OS Version updated
  - Old: Rocky Linux 9.2
  - New: Rocky Linux 9.3
  - Severity: High
  - Category: OS

[2025-11-06 10:30] Nginx version updated
  - Old: 1.23.0
  - New: 1.24.0
  - Severity: Medium
  - Category: Services

[2025-11-06 10:30] New feature enabled: Docker
  - Value: 24.0.7
  - Severity: Medium
  - Category: Features
```

## Monitoring in Admin Panel

### Real-time Status

**Dashboard view:**
- Total systems count
- Active / Inactive / Unknown breakdown
- Recent inventory changes
- Systems requiring attention

**System list:**
- Heartbeat status indicator (🟢🟡⚪)
- Last heartbeat time
- Last inventory time
- Change notifications

### Alerts (if configured)

Automatic alerts for:
- System goes offline (no heartbeat for 15+ minutes)
- Critical changes detected in inventory
- New system registered
- System version mismatch
- Security vulnerabilities detected

### System Health

**Health score based on:**
- Heartbeat reliability (% uptime)
- Inventory freshness
- Number of changes
- Critical issues count

## Troubleshooting

### Authentication Fails (HTTP 401)

**Problem:** "Invalid system credentials" or "Unauthorized"

**Solutions:**
1. Verify credentials are correct:
   ```bash
   echo -n "system_key:system_secret" | base64
   ```
2. Check for extra spaces in credentials
3. Ensure system is registered
4. Verify secret wasn't regenerated
5. Test with curl:
   ```bash
   curl -v -u "system_key:system_secret" \
     https://my.nethesis.it/collect/api/systems/heartbeat \
     -H "Content-Type: application/json" \
     -d '{}'
   ```

### Connection Timeout

**Problem:** Request times out, no response

**Solutions:**
1. Check network connectivity:
   ```bash
   ping my.nethesis.it
   ```
2. Verify port 8081 is accessible:
   ```bash
   telnet my.nethesis.it 8081
   ```
3. Check firewall rules (allow outbound to port 8081)
4. Verify DNS resolution
5. Test from different network

### Inventory Not Updating

**Problem:** Inventory sent successfully but not visible in admin panel

**Solutions:**
1. Wait 60 seconds and refresh (cache propagation)
2. Verify you're viewing the correct system
3. Check inventory was sent to correct endpoint (port 8081)
4. Verify system is not deleted
5. Check system logs for errors

### Heartbeat Shows as "Dead"

**Problem:** System shows red/dead status despite sending heartbeat

**Solutions:**
1. Check heartbeat frequency (must be < 15 minutes)
2. Verify heartbeat is reaching platform:
   ```bash
   curl -v https://my.nethesis.it/collect/api/systems/heartbeat \
     -u "key:secret" -H "Content-Type: application/json" -d '{}'
   ```
3. Check system time is synchronized (NTP)
4. Verify no clock drift
5. Review collect service logs (admin only)

### Changes Not Detected

**Problem:** Inventory sent but no changes shown

**Solutions:**
1. Verify actual data changed between inventories
2. Check that changed fields are supported
3. Small numerical changes may not trigger detection
4. Custom fields are tracked for changes
5. Wait for next inventory and compare

## Best Practices

### Heartbeat

- Send every 5 minutes consistently
- Use scheduled task (cron, systemd timer)
- Log heartbeat failures for debugging
- Implement retry logic (exponential backoff)
- Monitor heartbeat success rate

### Inventory

- Send complete inventory every time
- Don't send partial updates
- Include all relevant data
- Use consistent field names
- Validate JSON before sending
- Send immediately after significant changes

### Error Handling

- Implement retry logic for network failures
- Log all errors with context
- Don't retry authentication errors (401)
- Use exponential backoff for retries
- Alert on repeated failures

### Security

- Store credentials securely
- Never log credentials
- Use HTTPS only
- Verify SSL certificates
- Rotate credentials periodically
- Monitor for authentication failures

### Performance

- Compress large inventories
- Batch data collection
- Avoid unnecessary inventory sends
- Use efficient data structures
- Monitor network bandwidth

## Related Documentation

- [System Registration](05-system-registration.md)
- [Systems Management](04-systems.md)
- [Backend API](https://github.com/NethServer/my/blob/main/backend/README.md)
- [Collect Service](https://github.com/NethServer/my/blob/main/collect/README.md)
