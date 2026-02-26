# Mimir — Alerting Infrastructure

Grafana Mimir runs as a multi-tenant **Alertmanager** (`-target=alertmanager`) for the MY platform, deployed on a dedicated VM (Server B). It does **not** ingest metrics. The collect service on Server A routes alert notifications through Mimir's Alertmanager API.

## Topology

```
┌──────────────────────────────────┐      ┌──────────────────────────────────┐
│  Server A (main app)             │      │  Server B (alerting VM)          │
│                                  │      │                                  │
│  collect  ──/api/services/mimir──►──────►  mimir   (port 19009)           │
│  backend                         │      │    -target=alertmanager          │
│  frontend                        │      │      └── S3 alertmanager state   │
│  nginx proxy                     │      │                                  │
└──────────────────────────────────┘      └──────────────────────────────────┘
```

## Self-hosted deployment

All commands are run from the **repository root**.

**1. Configure environment variables**

```bash
cp services/mimir/.env.example services/mimir/.env
# Edit mimir/.env and fill in all required values
```

**2. Start the stack**

```bash
podman-compose -f services/mimir/docker-compose.yml up -d
```

This starts `mimir` on the `metrics-network` bridge.

**3. Verify**

```bash
curl http://localhost:19009/ready   # mimir
```

Should return `ready`.

**Service ports (host)**

| Service | Host port |
|---------|-----------|
| mimir   | 19009     |

## Environment variables

| Variable | Description | Example |
|----------|-------------|---------|
| `MIMIR_S3_ENDPOINT` | S3-compatible storage endpoint | `ams3.digitaloceanspaces.com` |
| `MIMIR_S3_ACCESS_KEY` | S3 access key | `your-access-key` |
| `MIMIR_S3_SECRET_KEY` | S3 secret key | `your-secret-key` |
| `MIMIR_S3_ALERTMANAGER_BUCKET` | Bucket for Alertmanager state | `my-mimir-alertmanager` |

Copy `services/mimir/.env.example` to `services/mimir/.env` and fill in every value before starting the stack.

## Architecture

Mimir runs as an alertmanager-only target (`-target=alertmanager`). It uses a single S3 bucket for persistent Alertmanager state. Multitenancy is enabled; all requests from `collect` include the tenant ID resolved from the system's organization.

The config template (`services/mimir/my.yaml`) uses `${VAR}` placeholders that are expanded at container startup by `entrypoint.sh` via `envsubst`.

## Render.com deployment

`render.yaml` defines two private services (`type: pserv`, not internet-accessible):

| Service name | Environment |
|---|---|
| `my-mimir-prod` | production |
| `my-mimir-qa`   | QA |

The collect service's `MIMIR_URL` is set to `http://my-mimir-prod:9009` (production) — traffic never leaves the private network.

S3 credentials must be added as Render environment secrets for each service.
