---
sidebar_position: 4
---

# Configuration Backups

Each registered system — NethServer 8 (NS8) and NethSecurity — uploads an encrypted snapshot of its configuration to MY on a daily schedule. This page describes what is stored, how the data is protected, and how operators interact with the backup subsystem.

## Overview

Backups are **end-to-end encrypted on the appliance** before they are uploaded. MY stores the ciphertext together with a short set of metadata (size, SHA-256, upload timestamp, appliance version) on an S3-compatible object store; the server never sees the plaintext of the backup and cannot decrypt it.

```
┌──────────────┐   GPG-encrypted blob     ┌──────────────┐    PutObject    ┌──────────────┐
│  Appliance   │ ───────────────────────► │   collect    │ ───────────────►│   S3 bucket  │
│ (NS8 / NSEC) │   POST /systems/backups  │  (ingest)    │                 │  (DO Spaces, │
└──────────────┘   HTTP Basic auth        └──────────────┘                 │   AWS S3, …) │
                                                                           └──────▲───────┘
                                                                                  │
                                                                                  │ presigned URL
                                                                                  │
                                                                          ┌──────────────┐
                                                                          │   backend    │
                                                                          │  (list/read) │
                                                                          └──────────────┘
```

## Authentication and access control

Uploads use **HTTP Basic auth** with the same `system_key:system_secret` pair the appliance already uses for inventory and heartbeat (see [system registration](registration)). A system can only write or read its own prefix on the bucket — cross-tenant access is refused server-side.

Reads performed by users go through `backend` with the regular Logto-issued JWT and the same RBAC rules that gate `GET /systems/:id`: a user sees a system's backups only if the user's organization owns that system.

## Storage layout

Every backup object is keyed as:

```
{org_id}/{system_key}/{backup_id}.{ext}
```

- `org_id` is the organization that owns the system.
- `system_key` is the stable user-facing identifier (`NETH-…`) the appliance authenticates with — chosen over the internal UUID so operators browsing a raw bucket listing can recognise each system at a glance.
- `backup_id` is a time-ordered UUIDv7 generated at upload time.
- `ext` reflects the compression / encryption format detected from the appliance-provided filename (`.tar.gz`, `.tar.xz`, `.gpg`, `.bin`).

Per-object metadata travels as standard `x-amz-meta-*` headers:

| Header                   | Meaning                                                                 |
|--------------------------|-------------------------------------------------------------------------|
| `x-amz-meta-sha256`      | SHA-256 of the encrypted blob, computed by `collect` during the stream. |
| `x-amz-meta-filename`    | User-facing name provided by the appliance (`X-Filename` header).       |
| `x-amz-meta-uploader-ip` | Peer address seen by `collect` at ingest — not forgeable via proxies.   |

## Retention and quotas

The ingest path enforces three independent caps per system:

| Setting                          | Default     | Meaning                                           |
|----------------------------------|-------------|---------------------------------------------------|
| `BACKUP_MAX_PER_SYSTEM`          | 10          | Maximum number of backups kept per system.        |
| `BACKUP_MAX_SIZE_PER_SYSTEM`     | 500&nbsp;MB | Maximum total bytes stored per system.            |
| `BACKUP_MAX_UPLOAD_SIZE`         | 2&nbsp;GB   | Hard limit on a single upload.                    |

A per-organization ceiling is also enforced:

| Setting                      | Default    | Meaning                                                   |
|------------------------------|------------|-----------------------------------------------------------|
| `BACKUP_MAX_SIZE_PER_ORG`    | 100&nbsp;GB | Total bytes across every system in the same organization. Set to `0` to disable (logged as a warning at startup). |

When either the count or size threshold is exceeded the oldest object under the system's prefix is pruned until the backup fits. Pruning is serialised by a Redis lock so concurrent uploads from the same appliance cannot race over the victim.

Appliance uploads are additionally rate-limited per system (default 6 per minute, 60 per hour). Legitimate NS8 and NethSecurity installations run on a daily timer and never hit these limits; the caps exist to contain flood-style abuse.

## Deletion and GDPR

A **soft delete** keeps the backups in place: the system row is flagged with `deleted_at` but can still be restored via the UI, and its backups must survive so the restore is useful. A **hard destroy** is irreversible and runs a GDPR-aligned erasure — every object under the system's `{org_id}/{system_key}/` prefix is removed from the bucket before the database row is dropped, whether the system was previously soft-deleted or not. If the system was previously reassigned across organizations, every prior `org_id` prefix is swept too, so a partial cleanup failure during a past reassignment cannot leak ciphertext past the destroy. If the storage cleanup fails the destroy is refused so the operator can retry; no orphan ciphertext is ever left behind under a destroyed system's prefix. Credential changes (secret rotation, soft delete) invalidate every cached auth entry on `collect` within a second through a cross-service Redis pub/sub bus.

## Cross-organization reassignment

When a system is moved from one organization to another, its backups
follow the new owner: everything under the previous owner's prefix is
copied to the new owner's prefix before the change is committed, and
the previous prefix is then cleared. The full mechanics — what carries
over, who can trigger it, what the previous owner sees after the
move — are documented in [Reassigning a system to another organization](org-reassignment).

## Managing backups

API endpoints exposed by `backend` for administrators:

| Method | Path                                                      | Purpose                                           |
|--------|-----------------------------------------------------------|---------------------------------------------------|
| `GET`  | `/api/systems/:id/backups`                                | List every backup for the system, with usage counters. |
| `GET`  | `/api/systems/:id/backups/:backup_id/download`            | Return a presigned download URL (5 min TTL).      |
| `DELETE` | `/api/systems/:id/backups/:backup_id`                   | Delete a specific backup.                          |

Presigned URLs are minted server-side and carry no authentication — treat them as short-lived bearer tokens and do not share them.

A UI for listing, downloading, and deleting backups lives under the system detail view.

## Related

- [System registration](registration) — how an appliance obtains the credentials used for backup uploads.
- [Reassigning a system to another organization](org-reassignment) — what happens to backups when a system changes owner.
- [`collect/README.md`](https://github.com/NethServer/my/blob/main/collect/README.md) — storage configuration (`BACKUP_S3_*`) and a copy-paste `curl` recipe for simulating an appliance upload.
- [`backend/README.md`](https://github.com/NethServer/my/blob/main/backend/README.md) — matching read-side storage configuration.
