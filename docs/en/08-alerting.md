# Alerting

Learn how My platform manages alert rules and sends notifications per organization using Grafana Mimir's multi-tenant Alertmanager.

## Overview

My platform uses [Grafana Mimir](https://grafana.com/oss/mimir/)'s built-in multi-tenant Alertmanager to manage alert rules and route notifications. Each organization has its own isolated Alertmanager configuration — alert rules and notification receivers (e.g. email, PagerDuty, webhook) are fully scoped to the organization that owns them.

## How It Works

### Multi-Tenancy

Each system belongs to an organization. The collect service resolves the system's `organization_id` from its credentials and injects it as the `X-Scope-OrgID` header before forwarding to Mimir. This ensures alert rules and notifications are fully isolated between organizations — each organization only manages and receives its own alerts.

## Authentication

Alertmanager API calls use the same credentials as system registration and inventory:

| Field | Value |
|-------|-------|
| **Username** | `system_key` (e.g. `NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE`) |
| **Password** | `system_secret` (e.g. `my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0`) |
| **Method** | HTTP Basic Auth |

No separate registration is needed — any system that has completed registration can immediately interact with the Alertmanager API. See [System Registration](05-system-registration.md) for how to obtain credentials.

## Alertmanager API

The collect service proxies Alertmanager API calls and automatically injects the `X-Scope-OrgID` header based on the authenticated system's organization. The base path is:

```
https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/
```

This maps directly to the [Alertmanager v2 API](https://github.com/prometheus/alertmanager/blob/main/api/v2/openapi.yaml). All standard endpoints are available.

### Example: List Active Alerts

```bash
curl https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/alerts \
  -u "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE:my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0"
```

### Example: Get Alertmanager Status

```bash
curl https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/status \
  -u "<system_key>:<system_secret>"
```

### Example: Create or Update a Silence

```bash
curl -X POST https://my.nethesis.it/api/services/mimir/alertmanager/api/v2/silences \
  -u "<system_key>:<system_secret>" \
  -H "Content-Type: application/json" \
  -d '{
    "matchers": [{"name": "alertname", "value": "WatchdogDown", "isRegex": false}],
    "startsAt": "2024-01-01T00:00:00Z",
    "endsAt": "2024-01-02T00:00:00Z",
    "createdBy": "admin",
    "comment": "Planned maintenance"
  }'
```

!!! tip
    Replace `<system_key>` and `<system_secret>` with the actual credentials stored on the system. The `X-Scope-OrgID` header is injected automatically by the collect service — do not set it manually.

## Troubleshooting

### HTTP 401 Unauthorized

**Cause:** Incorrect `system_key` or `system_secret`.

**Solutions:**
1. Verify credentials match what is stored on the system
2. Ensure the system has completed registration (see [System Registration](05-system-registration.md))
3. Check for leading/trailing spaces in the credentials

### HTTP 500 Internal Server Error

**Cause:** Mimir Alertmanager backend is unreachable or misconfigured.

**Solutions:**
1. This is a platform-side issue — contact your administrator
2. Check platform status page or monitoring alerts
3. Retry after a few minutes; Mimir may be restarting

## Related Documentation

- [System Registration](05-system-registration.md)
- [Inventory and Heartbeat](06-inventory-heartbeat.md)
- [Systems Management](04-systems.md)
