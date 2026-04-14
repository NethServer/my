# Backup — Local Development

This directory holds development-only tooling for the backup subsystem. Configuration backups are stored as S3 objects; production targets DigitalOcean Spaces (configured via `BACKUP_S3_*` on `collect` and `backend`). For local development [Garage](https://garagehq.deuxfleurs.fr) provides an S3-compatible endpoint backed by a local volume.

## Running Garage locally

From the repository root:

```bash
podman-compose -f services/backup/docker-compose.local.yml up -d
```

This starts:

| Service | Host port | Purpose |
|---------|-----------|---------|
| Garage S3 API | 13900 | S3 endpoint used by `collect` and `backend` |
| Garage Admin API | 13903 | Cluster admin + health check |

A bootstrap container (`garage-init`) runs once and assigns the cluster layout, creates the bucket `my-backups-dev`, and imports a fixed service key. All bootstrap commands are idempotent, so re-running the compose file does not duplicate resources.

Credentials created by the bootstrap (paste verbatim into `backend/.env` and `collect/.env`):

```
BACKUP_S3_ACCESS_KEY=backup-local-key
BACKUP_S3_SECRET_KEY=backup-local-secret-backup-local-secret-0000000000
```

## Wiring the local stack

**Standalone components running on the host** (e.g. `go run main.go`):

Add to `backend/.env` and `collect/.env`:

```
BACKUP_S3_ENDPOINT=http://localhost:13900
BACKUP_S3_REGION=garage
BACKUP_S3_BUCKET=my-backups-dev
BACKUP_S3_ACCESS_KEY=backup-local-key
BACKUP_S3_SECRET_KEY=backup-local-secret-backup-local-secret-0000000000
BACKUP_S3_USE_PATH_STYLE=true
```

**Full-stack running in containers** (root `docker-compose.yml`):

Backend and collect reach Garage via the shared compose network and use a different hostname than the browser. Join the `backup-local-network` network and set:

```
BACKUP_S3_ENDPOINT=http://garage:3900
BACKUP_S3_PRESIGN_ENDPOINT=http://localhost:13900
BACKUP_S3_USE_PATH_STYLE=true
```

`BACKUP_S3_PRESIGN_ENDPOINT` is backend-only. When set, the backend signs download URLs with this hostname instead of `BACKUP_S3_ENDPOINT`, so the browser can follow them without a DNS or signature mismatch. In production the variable stays unset and both roles use the same Spaces endpoint.

## Inspecting local storage

Garage ships no web UI; use the CLI via `podman exec`:

```bash
# Cluster state and bucket list
podman exec backup-local-garage /garage -c /etc/garage/garage.toml status
podman exec backup-local-garage /garage -c /etc/garage/garage.toml bucket list

# Bucket usage (object count, total bytes, per-key permissions)
podman exec backup-local-garage /garage -c /etc/garage/garage.toml bucket info my-backups-dev

# List objects under a system's prefix
podman exec backup-local-garage /garage -c /etc/garage/garage.toml bucket list-objects \
    my-backups-dev --prefix "<org_id>/<system_id>/"
```

Or with the AWS CLI, which speaks the S3 protocol directly:

```bash
aws --endpoint-url http://localhost:13900 \
    --region garage \
    s3 ls s3://my-backups-dev/ --recursive
```

(with `AWS_ACCESS_KEY_ID=backup-local-key` and the matching secret in the environment).

## Exercising the round-trip

1. Start the full app stack and Garage.
2. Create a system via the UI → save the `system_key` and `system_secret`.
3. Upload a backup as an appliance would:

   ```bash
   curl -u "$SYSTEM_KEY:$SYSTEM_SECRET" \
        -H "X-Filename: daily.tar.gz" \
        -H "Content-Type: application/gzip" \
        --data-binary @/path/to/backup.tar.gz \
        http://localhost:18081/api/systems/backups
   ```

4. Verify the object landed via `garage bucket list-objects` or the AWS CLI commands above.
5. As a logged-in user hit the backend list endpoint and then the download endpoint; follow the `download_url` it returns.

## Resetting local storage

```bash
podman-compose -f services/backup/docker-compose.local.yml down -v
```

The `-v` flag removes the `garage-meta` and `garage-data` volumes so the next start comes up with an empty cluster that the bootstrap container re-initialises.
