# Metrics

Learn how external systems push Prometheus metrics to My platform and how to visualize them in Grafana.

## Overview

My platform supports Prometheus metrics collection via [Grafana Mimir](https://grafana.com/oss/mimir/). Any registered NethServer or NethSecurity system can push metrics using the standard Prometheus `remote_write` protocol. Metrics are isolated per organization and visible in Grafana dashboards.

## How It Works

### Metrics Ingestion Flow

```
┌─────────────────┐                                      ┌──────────────┐
│  NethServer /   │  POST /mimir/api/v1/push              │              │
│  NethSecurity   │ ─────────────────────────────────>   │    nginx     │
│                 │  Basic Auth: system_key:system_secret │              │
└─────────────────┘                                      └──────┬───────┘
                                                                │
                                                                │ /api/mimir/
                                                                v
                                                         ┌──────────────┐
                                                         │   Backend    │
                                                         │              │
                                                         │  1. Validate │
                                                         │     Basic    │
                                                         │     Auth     │
                                                         │  2. Set      │
                                                         │  X-Scope-    │
                                                         │  OrgID:      │
                                                         │  <org_id>    │
                                                         └──────┬───────┘
                                                                │
                                                                v
                                                         ┌──────────────┐
                                                         │    Mimir     │
                                                         │  (private)   │
                                                         └──────────────┘
```

### Grafana Access Flow

```
┌─────────────┐                   ┌──────────────┐     ┌──────────────┐
│   Browser   │  /grafana/        │    nginx     │     │   Grafana    │
│             │ ─────────────>    │              │ --> │              │
│             │                   └──────────────┘     └──────┬───────┘
└─────────────┘                                               │
                                                              │ queries Mimir
                                                              │ X-Scope-OrgID
                                                              v
                                                       ┌──────────────┐
                                                       │    Mimir     │
                                                       │  (private)   │
                                                       └──────────────┘
```

### Multi-Tenancy

Each system belongs to an organization. The backend resolves the system's `organization_id` from its credentials and injects it as the `X-Scope-OrgID` header before forwarding to Mimir. This ensures metrics are fully isolated between organizations — each organization only sees its own data.

## Authentication

Metrics push uses the same credentials as system registration and inventory:

| Field | Value |
|-------|-------|
| **Username** | `system_key` (e.g. `NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE`) |
| **Password** | `system_secret` (e.g. `my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0`) |
| **Method** | HTTP Basic Auth |

No separate registration is needed — any system that has completed registration can immediately push metrics. See [System Registration](05-system-registration.md) for how to obtain credentials.

## Configuring Prometheus `remote_write`

Add the following block to your Prometheus configuration (`/etc/prometheus/prometheus.yml` or equivalent):

```yaml
remote_write:
  - url: https://my.nethesis.it/mimir/api/v1/push
    basic_auth:
      username: <system_key>
      password: <system_secret>
```

Replace `<system_key>` and `<system_secret>` with the actual credentials stored on the system.

**Example with real-looking values:**
```yaml
remote_write:
  - url: https://my.nethesis.it/mimir/api/v1/push
    basic_auth:
      username: NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE
      password: my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0
```

After updating the configuration, reload Prometheus:
```bash
systemctl reload prometheus
# or send SIGHUP
kill -HUP $(pidof prometheus)
```

!!! tip
    Prometheus will start forwarding all scraped metrics to My. Use `remote_write_queue_samples_total` in your local Prometheus to verify metrics are being sent.

## Accessing Grafana

Grafana is available at:

```
https://my.nethesis.it/grafana/
```

Dashboards are **per-organization**: each organization's users can only see metrics collected from systems belonging to their organization. The tenant isolation is enforced automatically via `X-Scope-OrgID`.

!!! note
    Grafana access is managed by your platform administrator. Contact them to get access or to request custom dashboards for your organization.

## Troubleshooting

### HTTP 401 Unauthorized

**Cause:** Incorrect `system_key` or `system_secret`.

**Solutions:**
1. Verify credentials match what is stored on the system
2. Ensure the system has completed registration (see [System Registration](05-system-registration.md))
3. Check for leading/trailing spaces in the credentials
4. Test manually:
   ```bash
   curl -X POST https://my.nethesis.it/mimir/api/v1/push \
     -u "NOC-F64B-A989-C9E7-45B9-A55D-59EC-6545-40EE:my_a1b2c3d4e5f6g7h8i9j0.k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0" \
     -H "Content-Type: application/x-protobuf" \
     --data-binary @/dev/null
   ```
   A `400 Bad Request` (not 401) confirms authentication is working.

### HTTP 500 Internal Server Error

**Cause:** Mimir backend is unreachable or misconfigured.

**Solutions:**
1. This is a platform-side issue — contact your administrator
2. Check platform status page or monitoring alerts
3. Retry after a few minutes; Mimir may be restarting

### Metrics Not Appearing in Grafana

**Cause:** Metrics are being sent but not yet visible.

**Solutions:**
1. Wait 1–2 minutes — Mimir has an ingestion delay
2. Verify `remote_write` is enabled in Prometheus and the configuration is correct
3. Check Prometheus logs for remote write errors:
   ```bash
   journalctl -u prometheus -n 50 | grep remote_write
   ```
4. Confirm you are logged in to Grafana with an account that belongs to the correct organization

## Related Documentation

- [System Registration](05-system-registration.md)
- [Inventory and Heartbeat](06-inventory-heartbeat.md)
- [Systems Management](04-systems.md)
