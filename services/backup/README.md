# Backup — Object Storage

[Garage](https://garagehq.deuxfleurs.fr) provides an S3-compatible object store for appliance configuration backups during local development. The production target is DigitalOcean Spaces (configured via `BACKUP_S3_*` on `collect` and `backend`); the compose file in this directory stands up a single-node Garage with local volumes so the round-trip (appliance upload → collect ingest → backend list/download) can be exercised without cloud credentials.

## Topology

```
appliance ──POST /api/systems/backups──► collect ──S3 PutObject──►  Garage
                                                                      │
backend   ◄─────────presigned URL──────────────────────────────── Garage
```

## Quick start

All commands are run from `services/backup/`.

```bash
# 1. Start Garage + bootstrap (auto-generates RPC/admin secrets in .env
#    if missing, then creates the bucket and fixed access key)
make dev-up

# 2. Append BACKUP_S3_* entries to backend/.env and collect/.env
make dev-setup

# 3. Restart backend and collect so they pick up the new env vars

# 4. Exercise the full appliance simulation
make test-roundtrip
```

The first `make dev-up` generates fresh `GARAGE_RPC_SECRET` and `GARAGE_ADMIN_TOKEN` values in `services/backup/.env` and leaves them there for subsequent runs. The admin port is bound to `127.0.0.1` only, so the token never leaves the local host.

See `make help` equivalents by listing the Makefile — every target starts with `dev-` (lifecycle) or `test-` (verification).

## Service ports (host)

| Service           | Host port | Container port | Purpose                       |
|-------------------|-----------|----------------|-------------------------------|
| Garage S3 API     | 13900     | 3900           | Used by collect and backend   |
| Garage Admin API  | 13903     | 3903           | Health checks and metrics     |

RPC (3901) stays container-internal — do not expose.

## Fixed local credentials

The bootstrap container (`garage-init`) is idempotent: every `make dev-up` converges on the same bucket and the same access key so `backend/.env` and `collect/.env` entries need to be written only once.

| Variable                    | Value                                                             |
|-----------------------------|-------------------------------------------------------------------|
| `BACKUP_S3_ENDPOINT`        | `http://localhost:13900`                                          |
| `BACKUP_S3_REGION`          | `garage`                                                          |
| `BACKUP_S3_BUCKET`          | `my-backups-dev`                                                  |
| `BACKUP_S3_ACCESS_KEY`      | `backup-local-key`                                                |
| `BACKUP_S3_SECRET_KEY`      | `backup-local-secret-backup-local-secret-0000000000`              |
| `BACKUP_S3_USE_PATH_STYLE`  | `true`                                                            |

`make dev-setup` writes the block above to `backend/.env` and `collect/.env` (skipping the file if the keys are already present).

**Containerised backend/collect**: when the components run under the root `docker-compose.yml` instead of directly on the host, reach Garage via the container-internal hostname and tell the backend to sign presigned URLs with the host-visible endpoint:

```
BACKUP_S3_ENDPOINT=http://garage:3900
BACKUP_S3_PRESIGN_ENDPOINT=http://localhost:13900
BACKUP_S3_USE_PATH_STYLE=true
```

## Inspecting storage

Garage ships no web UI. Use the CLI through `podman exec`:

```bash
# Cluster health + bucket list
make dev-status
make dev-objects            # bucket info for my-backups-dev

# List objects for a single system
podman exec backup-local-garage /garage -c /etc/garage/garage.toml \
    bucket list-objects my-backups-dev --prefix "<org_id>/<system_id>/"

# Or via awscli (path-style)
aws --endpoint-url http://localhost:13900 --region garage \
    s3 ls s3://my-backups-dev/ --recursive
```

## Object layout

S3 keys follow:

```
{org_id}/{system_id}/{backup_id}.{ext}
```

- `org_id` is the organization Logto ID — enables per-tenant lifecycle and quota.
- `system_id` is the internal `my` system UUID, stable across credential rotations.
- `backup_id` is a UUIDv7 generated at ingest time.
- `ext` is derived from a compound-aware filename parser (`.tar.gz`, `.tar.xz`, `.gpg`, defaults to `.bin`).

Per-object metadata is stored as S3 headers (`x-amz-meta-*`):

| Header                   | Source                                                              |
|--------------------------|---------------------------------------------------------------------|
| `x-amz-meta-sha256`      | Computed by `collect` via streaming tee at ingest                   |
| `x-amz-meta-filename`    | From appliance request header `X-Filename`                          |
| `x-amz-meta-uploader-ip` | Client IP observed by `collect`                                     |
| `x-amz-meta-uploader-ua` | Appliance `User-Agent`                                              |
| `x-amz-meta-system-ver`  | Optional appliance OS version from `X-System-Version`               |

## Retention

Enforced by `collect` after each upload by listing the system's prefix and pruning the oldest objects if either limit is exceeded:

| Knob                            | Default           |
|---------------------------------|-------------------|
| `BACKUP_MAX_PER_SYSTEM`         | 10 objects        |
| `BACKUP_MAX_SIZE_PER_SYSTEM`    | 500 MB            |
| `BACKUP_MAX_UPLOAD_SIZE`        | 2 GB (per upload) |

A Garage-level lifecycle rule can complement the inline enforcement by deleting anything older than `N` days; configure with `garage bucket lifecycle add`.

## Exercising the round-trip

`test-roundtrip.sh` reproduces the NS8 / NethSecurity upload pipeline against the locally running collect endpoint: build a minimal JSON payload, gzip it, GPG-symmetric-encrypt it, and POST it with HTTP Basic auth.

```bash
export SYSTEM_KEY="my_sys_XXXXXXXXXXXXXXXX"
export SYSTEM_SECRET="my_XXXXXXXXXXXXXXXXXXXX.YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY"
export COLLECT_URL="http://localhost:8081"
make test-roundtrip
```

## Resetting local storage

```bash
make dev-reset
```

Removes the `garage-meta` and `garage-data` volumes so the next `make dev-up` converges on an empty cluster that the bootstrap container re-initialises.

## Production deployment

The Render blueprint provisions no Garage instance — production backups land in DigitalOcean Spaces via `BACKUP_S3_*` on `my-backend-prod` / `my-backend-qa` and `my-collect-prod` / `my-collect-qa`. The `Containerfile` in this directory is kept for parity with other `services/` entries and for ad-hoc self-hosted deployments.
