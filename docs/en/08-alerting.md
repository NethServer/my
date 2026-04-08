# Alerting

Learn how the My platform sends and manages alerts per organization using Grafana Mimir's multi-tenant Alertmanager.

## Overview

My platform uses [Grafana Mimir](https://grafana.com/oss/mimir/)'s built-in multi-tenant Alertmanager. Each system belongs to an organization — the collect service resolves the system's `organization_id` from its credentials and injects it as the `X-Scope-OrgID` header before forwarding to Mimir, ensuring alerts are fully isolated between organizations.

For complete API documentation, see the [Prometheus Alertmanager v2 OpenAPI Specification](https://github.com/prometheus/alertmanager/blob/main/api/v2/openapi.yaml).

## Labels and Annotations Convention

All alerts sent to My platform must follow this convention to ensure correct routing, display, and notification.

### Labels

| Label | Required | Description |
|-------|----------|-------------|
| `alertname` | ✅ Required | CamelCase identifier for the alert type. Use descriptive names, e.g. `DiskFull`, `WanDown`, `BackupFailed`. |
| `severity` | ✅ Required | Alert severity level. Allowed values: `critical`, `warning`, `info`. Defaults to `info` if not set. |
| `system_key` | ✅ Auto-added | Automatically injected by the collect service from the authenticated system's credentials. Do not set manually. |
| `service` | Optional | Sub-service within the system that generated the alert, e.g. `backup`, `storage`, `ha`, `network`. |

**`alertname` format:** Use UpperCamelCase (PascalCase). Each word starts with a capital letter with no separators. Examples: `DiskFull`, `WanDown`, `HaSyncFailed`, `CertExpired`.

**`severity` levels:**

| Level | When to use |
|-------|-------------|
| `critical` | System is down or data loss is imminent. Requires immediate action. |
| `warning` | Degraded state or approaching a threshold. Action needed soon. |
| `info` | Informational event. No immediate action required. |

### Annotations

| Annotation | Required | Description |
|------------|----------|-------------|
| `description_en` | ✅ Required | Human-readable English description of the alert condition. |
| `description_it` | Optional | Italian translation of the description. |

### Example alert payload

```json
[{
  "labels": {
    "alertname": "DiskFull",
    "severity": "critical",
    "service": "storage"
  },
  "annotations": {
    "description_en": "Disk usage on /data is above 90%. Free space: 4 GB.",
    "description_it": "Utilizzo del disco su /data superiore al 90%. Spazio libero: 4 GB."
  },
  "startsAt": "2026-01-15T10:30:00Z",
  "endsAt": "0001-01-01T00:00:00Z"
}]
```

> `system_key` is injected automatically — do not include it in the payload.

---

## Alert Catalog

Standard alert types defined for NethServer and NethSecurity systems. Alerts are unified across products to simplify maintenance and integration.

### Storage & System

| `alertname` | Severity | Service | `description_en` |
|-------------|----------|---------|-----------------|
| `DiskSpaceLow` | `warning` | `storage` | Disk usage on `{{ $labels.mountpoint }}` is above 80%. Free space is running low. |
| `DiskSpaceCritical` | `critical` | `storage` | Disk usage on `{{ $labels.mountpoint }}` is above 90%. Immediate action required. |
| `SwapFull` | `warning` | — | Swap space is filling up (usage above 80%). Current value: `{{ $value }}`. |
| `SwapNotPresent` | `critical` | — | Swap is not configured on this host. |
| `RaidDiskFailed` | `critical` | `storage` | Software RAID array `{{ $labels.device }}` has a failed disk. Immediate attention required. |
| `RaidDriveMissing` | `critical` | `storage` | Software RAID array `{{ $labels.device }}` has insufficient active drives. |

### Infrastructure & Cluster

| `alertname` | Severity | Service | `description_en` |
|-------------|----------|---------|-----------------|
| `NodeOffline` | `critical` | — | Cluster node `{{ $labels.node }}` is offline. This may be caused by a network outage or a crashed metric exporter. |
| `LokiOffline` | `warning` | `loki` | Loki instance `{{ $labels.instance }}` is down or not running properly. |
| `WanDown` | `critical` | `network` | WAN interface `{{ $labels.interface }}` is down. Internet connectivity lost. |
| `HaSyncFailed` | `critical` | `ha` | High-availability synchronization between primary and secondary node has failed. |
| `HaPrimaryFailed` | `critical` | `ha` | High-availability primary node has failed. Failover may have occurred. |

### Certificates & Data Protection

| `alertname` | Severity | Service | `description_en` |
|-------------|----------|---------|-----------------|
| `CertExpiringSoon` | `warning` | — | TLS certificate `{{ $labels.cn }}` expires in less than 28 days (`{{ $value \| humanizeDuration }}`). |
| `CertExpiringCritical` | `critical` | — | TLS certificate `{{ $labels.cn }}` expires in less than 7 days (`{{ $value \| humanizeDuration }}`). |
| `CertExpired` | `critical` | — | TLS certificate `{{ $labels.cn }}` has expired (`{{ $value \| humanizeDuration }}` ago). |
| `BackupFailed` | `critical` | `backup` | Backup job `{{ $labels.name }}` (ID: `{{ $labels.id }}`) has failed. |
| `ConfigBackupNotEncrypted` | `warning` | `backup` | Configuration backup is not encrypted. Sensitive data may be exposed. |

## Authentication

Alertmanager API calls use the same credentials as system registration and inventory:

| Field | Value |
|-------|-------|
| **Username** | `system_key` (e.g. `NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE`) |
| **Password** | `system_secret` (e.g. `my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0`) |
| **Method** | HTTP Basic Auth |

No separate registration is needed — any system that has completed registration can immediately interact with the Alertmanager API. See [System Registration](05-system-registration.md) for how to obtain credentials.

## Alertmanager API

The collect service proxies Alertmanager API calls and automatically injects the `X-Scope-OrgID` header based on the authenticated system's organization.

| Use Case | Path |
|----------|------|
| **Alerts** | `/api/services/mimir/alertmanager/api/v2/alerts` |
| **Silences** | `/api/services/mimir/alertmanager/api/v2/silences[/{silence_id}]` |

## Common Examples

### 1. Alert Management

#### Inject an alert directly (Injection API)

```bash
curl -X POST \
  -u "system_key:system_secret" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts \
  -d '[{
    "labels": {
      "alertname": "DiskFull",
      "severity": "critical",
      "service": "storage"
    },
    "annotations": {
      "description_en": "Disk usage on /data is above 90%. Free space: 4 GB.",
      "description_it": "Utilizzo del disco su /data superiore al 90%. Spazio libero: 4 GB."
    },
    "generatorURL": "https://prometheus.your-domain.com/graph",
    "startsAt": "2024-01-15T10:30:00Z",
    "endsAt": "0001-01-01T00:00:00Z"
  }]'
```

**Response (200 OK)** - Alert successfully injected

**Note on resolution:** Setting `endsAt` to `0001-01-01T00:00:00Z` means the alert remains active indefinitely until explicitly resolved.

#### List active alerts

```bash
curl -u "system_key:system_secret" \
  -H "Accept: application/json" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts
```

**Response (200 OK):**
```json
[
  {
    "labels": {
      "alertname": "DiskFull",
      "severity": "critical",
      "service": "storage",
      "system_key": "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE"
    },
    "annotations": {
      "description_en": "Disk usage on /data is above 90%. Free space: 4 GB.",
      "description_it": "Utilizzo del disco su /data superiore al 90%. Spazio libero: 4 GB."
    },
    "startsAt": "2024-01-15T10:30:00Z",
    "endsAt": "0001-01-01T00:00:00Z",
    "generatorURL": "https://prometheus.your-domain.com/graph",
    "status": {
      "state": "active",
      "silencedBy": [],
      "inhibitedBy": []
    }
  }
]
```

#### Resolve an alert

To resolve an alert, send the same alert with `endsAt` set to a past timestamp:

```bash
curl -X POST \
  -u "system_key:system_secret" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts \
  -d '[{
    "labels": {
      "alertname": "DiskFull",
      "severity": "critical",
      "service": "storage"
    },
    "annotations": {
      "description_en": "Disk usage on /data is back to normal."
    },
    "generatorURL": "https://prometheus.your-domain.com/graph",
    "startsAt": "2024-01-15T10:30:00Z",
    "endsAt": "2024-01-15T11:30:00Z"
  }]'
```

---

### 2. Silence Management

#### Create a silence

```bash
curl -X POST \
  -u "system_key:system_secret" \
  -H "Content-Type: application/json" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/silences \
  -d '{
    "matchers": [
      {
        "name": "alertname",
        "value": "DiskFull",
        "isRegex": false
      },
      {
        "name": "service",
        "value": "storage",
        "isRegex": false
      }
    ],
    "startsAt": "2024-01-15T10:00:00Z",
    "endsAt": "2024-01-15T18:00:00Z",
    "createdBy": "admin@your-domain.com",
    "comment": "Planned maintenance window"
  }'
```

**Response (200 OK):**
```json
{
  "silenceID": "2b05304b-a71e-48c0-a877-bb4824e84969"
}
```

#### List active silences

```bash
curl -u "system_key:system_secret" \
  -H "Accept: application/json" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/silences
```

**Response (200 OK):** List of all active silences and their configurations.

#### Delete a silence

```bash
curl -X DELETE \
  -u "system_key:system_secret" \
  https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/silence/2b05304b-a71e-48c0-a877-bb4824e84969
```

**Response (200 OK)** - Silence deleted

## Troubleshooting

### HTTP 401 Unauthorized

**Cause:** Incorrect `system_key` or `system_secret`.

**Solutions:**
1. Verify credentials match what is stored on the system
2. Ensure the system has completed registration (see [System Registration](05-system-registration.md))
3. Check for leading/trailing spaces in the credentials
4. Test manually:
   ```bash
   curl https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts \
     -u "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE:my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
   ```
   A `200 OK` or `404 Not Found` response (not `401`) confirms authentication is working.

### HTTP 500 Internal Server Error

**Cause:** Mimir Alertmanager backend is unreachable or misconfigured.

**Solutions:**
1. This is a platform-side issue — contact your administrator
2. Check platform status page or monitoring alerts
3. Retry after a few minutes; Mimir may be restarting

### HTTP 400 Bad Request

**Cause:** Request body is invalid (malformed JSON, missing required fields, etc.)

**Solutions:**
1. Verify JSON is valid using an online tool like [jsonlint.com](https://www.jsonlint.com/)
2. Ensure all required fields are present
3. Check ISO 8601 date format (e.g. `2024-01-15T10:30:00Z`)

## Related Documentation

- [System Registration](05-system-registration.md)
- [Inventory and Heartbeat](06-inventory-heartbeat.md)
- [Systems Management](04-systems.md)
- [Mimir HTTP API Documentation](https://grafana.com/docs/mimir/latest/references/http-api/)
- [Prometheus Alertmanager v2 OpenAPI](https://github.com/prometheus/alertmanager/blob/main/api/v2/openapi.yaml)
