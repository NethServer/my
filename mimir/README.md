# Mimir — Metrics Infrastructure

Grafana Mimir provides long-term metrics storage for the MY platform, deployed as a 2-node cluster on a dedicated VM (Server B). Grafana is co-located on the same VM and queries Mimir through a proxy on the main application backend (Server A).

## Topology

```
┌──────────────────────────────────┐     ┌──────────────────────────────────┐
│  Server A (main app)             │     │  Server B (metrics VM)           │
│                                  │     │                                  │
│  backend  ──/api/mimir──►        │◄────│  grafana  (port 13000)           │
│  collect                         │     │                                  │
│  frontend                        │     │  mimir1   (port 19009)  ◄──┐     │
│  nginx proxy                     │     │  mimir2   (port 19010)  ◄──┘     │
│                                  │     │      └── memberlist gossip       │
└──────────────────────────────────┘     │      └── shared S3 storage       │
                                         └──────────────────────────────────┘
```

Grafana **does not** connect directly to Mimir. It queries `http://<BACKEND_SERVICE_NAME>/api/mimir` using Basic Auth (`MIMIR_SYSTEM_KEY:MIMIR_SYSTEM_SECRET`). The backend adds the required `X-Scope-OrgID` header before forwarding to Mimir.

## Self-hosted deployment

All commands are run from the **repository root**.

**1. Configure environment variables**

```bash
cp mimir/.env.example mimir/.env
# Edit mimir/.env and fill in all required values
```

**2. Start the stack**

```bash
docker-compose -f mimir/docker-compose.yml up -d
```

This starts `mimir1`, `mimir2`, and `grafana` on the `metrics-network` bridge.

**3. Verify**

```bash
curl http://localhost:19009/ready   # mimir1
curl http://localhost:19010/ready   # mimir2
```

Both should return `ready`.

**Service ports (host)**

| Service | Host port |
|---------|-----------|
| mimir1  | 19009     |
| mimir2  | 19010     |
| grafana | 13000     |

## Environment variables

| Variable | Description | Example |
|----------|-------------|---------|
| `MIMIR_S3_ENDPOINT` | S3-compatible storage endpoint | `ams3.digitaloceanspaces.com` |
| `MIMIR_S3_ACCESS_KEY` | S3 access key | `your-access-key` |
| `MIMIR_S3_SECRET_KEY` | S3 secret key | `your-secret-key` |
| `MIMIR_S3_BUCKET` | Bucket for blocks (TSDB chunks) | `my-mimir-blocks` |
| `MIMIR_S3_ALERTMANAGER_BUCKET` | Bucket for Alertmanager state | `my-mimir-alertmanager` |
| `MIMIR_S3_RULER_BUCKET` | Bucket for recording/alert rules | `my-mimir-ruler` |
| `MIMIR_SYSTEM_KEY` | Basic Auth username for Grafana → backend | `nss_yoursystemkey` |
| `MIMIR_SYSTEM_SECRET` | Basic Auth password for Grafana → backend | `my_pub.secret` |
| `BACKEND_SERVICE_NAME` | Hostname (and optional port) of Server A | `my.nethesis.it` |

Copy `mimir/.env.example` to `mimir/.env` and fill in every value before starting the stack.

## Cluster architecture

The two Mimir nodes form a cluster via **memberlist gossip** on port 7946. Both nodes share the same three S3 buckets (blocks, alertmanager, ruler) and are configured with `replication_factor: 2`, so every series is stored on both nodes. Multitenancy is enabled; all writes from `collect` include the tenant ID supplied by the backend.

The config template (`mimir/my.yaml`) uses `${VAR}` placeholders that are expanded at container startup by `entrypoint.sh` via `envsubst`.

## Render.com deployment

`render.yaml` defines four private services (`type: pserv`, not internet-accessible):

| Service name | Environment |
|---|---|
| `my-mimir1-prod` | production |
| `my-mimir2-prod` | production |
| `my-mimir1-qa`   | QA |
| `my-mimir2-qa`   | QA |

`MIMIR_JOIN_MEMBER1` and `MIMIR_JOIN_MEMBER2` are already set to the two service names for each environment. The backend's `MIMIR_URL` is set to `http://my-mimir1-prod:9009` (production) — traffic never leaves the private network.

S3 credentials and Grafana auth vars must be added as Render environment secrets for each service group.

## Grafana connection

Grafana does **not** talk directly to Mimir. The datasource template (`grafana/provisioning/datasources/mimir.yaml.template`) points to:

```
http://<BACKEND_SERVICE_NAME>/api/mimir
```

with Basic Auth credentials `MIMIR_SYSTEM_KEY` / `MIMIR_SYSTEM_SECRET`. Set `BACKEND_SERVICE_NAME` to the hostname (and port if non-standard) of Server A, for example `my.nethesis.it` or `192.168.1.10:8080`.

The backend proxy authenticates the request, resolves the tenant, and forwards it to the internal Mimir cluster with the `X-Scope-OrgID` header.
