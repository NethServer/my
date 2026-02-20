# Mimir — Metrics Infrastructure

Grafana Mimir provides long-term metrics storage for the MY platform, deployed as a single node on a dedicated VM (Server B). The collect service on Server A writes metrics to Mimir and proxies read queries.

## Topology

```
┌──────────────────────────────────┐      ┌──────────────────────────────────┐
│  Server A (main app)             │      │  Server B (metrics VM)           │
│                                  │      │                                  │
│  collect  ──/api/services/mimir──►│◄────│  mimir   (port 19009)           │
│  backend                         │      │      └── S3 storage              │
│  frontend                        │      │                                  │
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
| `MIMIR_S3_BUCKET` | Bucket for blocks (TSDB chunks) | `my-mimir-blocks` |
| `MIMIR_S3_ALERTMANAGER_BUCKET` | Bucket for Alertmanager state | `my-mimir-alertmanager` |
| `MIMIR_S3_RULER_BUCKET` | Bucket for recording/alert rules | `my-mimir-ruler` |

Copy `services/mimir/.env.example` to `services/mimir/.env` and fill in every value before starting the stack.

## Architecture

Mimir runs as a single node with `replication_factor: 1`. It uses three S3 buckets (blocks, alertmanager, ruler) for persistent storage. Multitenancy is enabled; all writes from `collect` include the tenant ID resolved from the system's organization.

The config template (`services/mimir/my.yaml`) uses `${VAR}` placeholders that are expanded at container startup by `entrypoint.sh` via `envsubst`.

## Render.com deployment

`render.yaml` defines two private services (`type: pserv`, not internet-accessible):

| Service name | Environment |
|---|---|
| `my-mimir-prod` | production |
| `my-mimir-qa`   | QA |

The collect service's `MIMIR_URL` is set to `http://my-mimir-prod:9009` (production) — traffic never leaves the private network.

S3 credentials must be added as Render environment secrets for each service.
