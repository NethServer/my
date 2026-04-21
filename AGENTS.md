# MY (my.nethesis.it)

Centralized authentication and management platform. Logto as IdP, RBAC with business hierarchy (Owner > Distributor > Reseller > Customer) and technical user roles (Super Admin, Admin, Support).

**Version**: v0.5.0 (pre-production). Canonical source: `version.json`.

---

## 1. Critical Rules

### 1.1 Pre-production mindset

No users in production. Forbidden language in code, comments, docs:
- Temporal: "was", "previously", "before", "has been", "used to"
- Migration-flavored: "refactored", "migrated from", "deprecated", "legacy", "backward compatibility"

The system IS designed the way it currently looks. Treat each change as the current design, not a migration. Remove temporal language on sight. No compatibility shims. Rule removed post-GA.

### 1.2 Post-change verification (mandatory)

After any code change, run the component's pre-commit before committing:

```bash
cd backend && make pre-commit      # fmt + lint + test + validate-docs (redocly)
cd collect && make pre-commit      # fmt + lint + test
cd sync && make pre-commit         # fmt + lint + test
cd frontend && npm run pre-commit  # format + lint + type-check + test + build
```

Security updates: handled automatically by Dependabot/Renovate — do not chase them manually.

---

## 2. Architecture

### 2.1 Components (actual state of this branch)

```
my/
  backend/        Go REST API (:8080). Gin, JWT, Logto integration. Main control plane.
  collect/        Go inventory + Mimir proxy (:8081). Worker pool, Redis queues, HTTP Basic auth.
  sync/           Go CLI (Cobra). Logto RBAC/org initialization and pull.
  frontend/       Vue 3 + TS SPA. Vite, Tailwind, Pinia.
  proxy/          nginx reverse proxy routing to backend/collect/frontend.
  services/mimir/ Grafana Mimir single-node + multi-tenant Alertmanager. S3 backend.
  services/support/    Artifacts only on this branch (prebuilt tunnel-client, examples). Source lives elsewhere.
  services/ssh-gateway/ Placeholder only on this branch (.env + host key). Not implemented here.
  docs/           Docusaurus site (published docs).
  presentation/   Slides and mockups (out of scope for code work).
```

The six first-class components (tracked in `version.json`): backend, collect, sync, frontend, proxy, services/mimir. Do not attempt to build/run support or ssh-gateway from this branch.

### 2.2 Authentication flow

```
Frontend --[Logto access_token]--> POST /api/auth/exchange
Backend validates token, fetches roles/permissions from Logto Management API
Backend returns custom JWT (24h access + 7d refresh) with embedded permissions
Frontend uses custom JWT for subsequent calls
```

Custom JWT claims: user_id, user_roles, user_permissions, org_role, org_permissions, organization_id, plus `impersonated_by` when acting as another user.

### 2.3 Impersonation

Owner-only. `POST /api/auth/impersonate` mints a 1h JWT with the target user's permissions. Requires the target to have opted-in via `POST /api/auth/impersonate/consent` (consent can be revoked with DELETE). All sessions and actions are audited via `impersonation_audit` middleware. No self-impersonation, no chaining.

---

## 3. Components

### 3.1 Backend (`backend/`)

Layered request flow:
```
main.go (routes + middleware chains)
  --> middleware/ (logto.go, jwt.go, rbac.go, impersonation_audit.go, rate_limit.go, self_modification.go)
  --> methods/ (handlers — one or more files per resource)
  --> services/ (logto/, local/, alerting/, csvimport/, email/, export/)
  --> entities/ (raw SQL, no ORM)
```

Notable backend `services/`:
- `services/logto/` — Logto Management API client
- `services/local/` — DB-backed domain services (users, systems, applications, etc.)
- `services/alerting/` — Mimir Alertmanager client + YAML config renderer + email templates (en/it, firing/resolved)
- `services/csvimport/` — CSV parsing for bulk import flows
- `services/email/` — transactional email dispatch
- `services/export/` — CSV/PDF export generation

Notable `methods/` groupings:
- Resource CRUD: `distributors.go`, `resellers.go`, `customers.go`, `users.go`, `systems.go`, `applications.go`
- Bulk import/export: `*_import.go`, `*_export.go`, `import_helpers.go`
- Alerting: `alerting.go` (backend-side — per-tenant config, active alerts from Mimir, history from DB)
- Backups: `backups.go` (list/download/delete of appliance configuration backups stored on S3; purges the system's prefix on hard delete for GDPR)
- Filters: `systems_filters.go`, `users_filters.go` — aggregation endpoints for UI filters
- Other: `auth.go`, `impersonate.go`, `rebranding.go`, `inventory.go`, `organizations.go`, `roles.go`, `totals.go`, `validators/`

Source of truth for routes: `backend/main.go`. Source of truth for the API contract: `backend/openapi.yaml` (validated by `make validate-docs`; edits to handlers require corresponding OpenAPI updates).

### 3.2 Collect (`collect/`)

Inventory ingestion + Mimir proxy + LinkFailed cron.

```
main.go
  --> middleware/ (auth.go HTTP Basic with SHA256; webhook_auth.go Bearer token)
  --> methods/ (inventory.go, heartbeat.go, system_info.go, rebranding.go,
                mimir.go   — reverse proxy to Mimir Alertmanager with X-Scope-OrgID injection
                alertmanager.go — /api/alert_history webhook receiver)
  --> workers/ (InventoryWorker, DiffWorker, NotificationWorker, CleanupWorker,
                QueueMonitorWorker, DelayedMessageWorker — all started by manager.go)
  --> differ/ (YAML-configured JSON diff engine, severity/significance)
  --> cron/ (heartbeat_monitor.go — alive/dead/zombie + LinkFailed alert poster)
```

Key properties:
- Systems auth with HTTP Basic (`system_key:system_secret`, SHA256 in DB).
- `/api/services/mimir/alertmanager/api/v2/{alerts,silences}[/*subpath]` proxied to Mimir with server-set `X-Scope-OrgID` and authoritative identity labels (`injectLabels` overwrites client values, strips when DB is NULL).
- `/api/alert_history` receives Alertmanager resolved-alert webhooks with Bearer auth (constant-time compare, fail-closed). `organization_id` is resolved at write-time from `systems.system_key`; unknown keys are dropped.
- `/api/systems/backups` ingests GPG-encrypted configuration backups from appliances. Stream body → S3 with SHA-256 `io.TeeReader`, metadata reconciled via same-key `CopyObject`, retention enforced inline under a Redis `SET NX` lock, per-system rate limit. Keys: `{org_id}/{system_id}/{backup_id}.{ext}`. Storage is any S3-compatible bucket configured via `BACKUP_S3_*` env vars (see `collect/README.md`).

### 3.3 Sync (`sync/`)

Cobra CLI. Commands: `init`, `sync`, `pull`, `prune`. Drives Logto setup and keeps local DB in sync.

```
cmd/sync/main.go
  --> internal/cli/         (cobra commands)
  --> internal/client/      (Logto API client, fetchAllPages[T], auth refresh)
  --> internal/config/      (YAML config loading — configs/config.yml)
  --> internal/sync/        (push engine: Config -> Logto)
        engine.go           orchestration
        roles.go / organization.go / resources.go / applications.go
        pull_engine.go      Logto -> local DB
```

`IsSystemEntityByPatterns` detects Logto system entities to skip. `upsertOrganizationEntity` handles distributor/reseller/customer upserts uniformly. Dry-run available on all commands.

### 3.4 Frontend (`frontend/`)

Vue 3 + TypeScript, Vite, Tailwind, Pinia. Alerting UI present (`src/views/AlertingView.vue`, `src/queries/alerting/`, `src/lib/alerting.ts`). Pre-commit includes build step to catch TS errors and Vite misconfigs.

### 3.5 Mimir (`services/mimir/`)

Single-node Grafana Mimir with S3-compatible backend and multi-tenant Alertmanager. Containerfile, Makefile, docker-compose.yml + docker-compose.local.yml. `scripts/` contains Python helpers (`alert.py`, `alerting_config.py`) for manual testing.

**Alerting integration**:
- Backend (`backend/services/alerting/`) renders Alertmanager YAML from `AlertingConfig` models and pushes via `POST /api/v1/alerts` per tenant. Email templates are Go `html/template`-embedded, en/it locales, firing + resolved variants.
- Collect proxies systems to Alertmanager `alerts`/`silences` with `X-Scope-OrgID` from the authenticated system's org.
- Alertmanager webhooks resolved alerts back to collect `/api/alert_history`, which persists them scoped by `organization_id` (column on `alert_history`, populated from the DB via `system_key` lookup — never trusted from the payload).

### 3.6 Proxy (`proxy/`)

nginx. Routes `/api/*` (backend), `/api/services/mimir/*` and `/api/alert_history` (collect), everything else to frontend. Mixed TLS certs under `my.localtest.me+*.pem` for local dev.

---

## 4. Authorization & RBAC

### 4.1 Hierarchy (org_role, case-insensitive; **always lowercase in code switches**)

```
Owner (Nethesis) > Distributors > Resellers > Customers
```

- **Owner**: full control, can target any org via `?organization_id=X`
- **Distributor**: manages own resellers + their customers
- **Reseller**: manages own customers
- **Customer**: read-only on own data; most alerting endpoints auto-pin to `user.OrganizationID` regardless of query params

### 4.2 User roles (technical capability)

- **Super Admin** — Owner org only, full platform admin
- **Admin** — system/user management, dangerous operations
- **Support** — standard operations, read-focused

### 4.3 Effective permissions

```
effective = org_permissions (from org_role) UNION user_permissions (from user_roles)
```

Embedded in the custom JWT at exchange time. No external calls during request handling.

### 4.4 Route protection

- `middleware.RequirePermission("read:systems")` — single permission gate
- `middleware.RequireResourcePermission("systems")` — HTTP-verb-aware (read on GET, manage on POST/PUT/PATCH, destroy on DELETE)
- `middleware.PreventSelfModification()` — blocks a user from acting on their own account for dangerous verbs
- Hierarchy check for cross-org reads/writes: `local.UserService.IsOrganizationInHierarchy(orgRole, userOrgID, targetOrgID)`
- RBAC filtering in SQL: `helpers.AppendOrgFilter(query, orgRole, orgID, tableAlias, args, nextArgIdx)` uses `GetAllowedOrgIDsForFilter` (cached org-ID set) for non-owner roles

---

## 5. Shared Infrastructure

PostgreSQL and Redis are shared by backend and collect. **Use `podman` locally, not `docker`.**

| Resource  | Container     | Port | Connection                                                              |
| --------- | ------------- | ---- | ----------------------------------------------------------------------- |
| Postgres  | `my-postgres` | 5432 | `postgresql://noc_user:noc_password@localhost:5432/noc?sslmode=disable` |
| Redis     | `my-redis`    | 6379 | `redis://localhost:6379`                                                |

Makefile shortcuts (from any Go component dir):
```bash
make dev-up    # starts postgres + redis
make dev-down
make db-migrate         # applies all pending backend migrations
make db-migration MIGRATION=019 ACTION=apply|rollback|status
make redis-flush
make redis-cli
```

Direct DB access: `podman exec -it my-postgres psql -U noc_user -d noc`.

---

## 6. Coding Patterns

### 6.1 Entity IDs

Entities synced with Logto (orgs, users) carry both `id` (DB UUID) and `logto_id` (Logto's ID).

- **URL params** (`:id`): use `logto_id` for users/distributors/resellers/customers. Use DB UUID for systems/applications.
- **Request body fields**: use `logto_id` when assigning to an organization.
- **Response**: always return BOTH `id` and `logto_id` (including nested org objects).
- **SQL JOINs** on `organization_id`: match both formats:
  ```sql
  ON (t.organization_id = org.logto_id OR t.organization_id = org.id::text)
  ```

### 6.2 Error messages

User-facing errors are **lowercase**. Example: `"the user has revoked consent for impersonation. please exit impersonation mode."`

### 6.3 List response format

```json
{
  "code": 200,
  "message": "<resource> retrieved successfully",
  "data": {
    "<plural_resource>": [...],
    "pagination": { "page", "page_size", "total_count", "total_pages",
                    "has_next", "has_prev", "sort_by", "sort_direction" }
  }
}
```

Build pagination with `helpers.BuildPaginationInfoWithSorting(...)`. Reference `$ref: '#/components/schemas/Pagination'` in OpenAPI.

### 6.4 SQL conventions

- Raw SQL, no ORM. Use `database.DB.QueryRow/Query/Exec`.
- Parameterized queries always (`$1`, `$2`). Never string-concatenate user input.
- For dynamic `ORDER BY`: allowlist the column name before use (see `entities/local_alertmanager_history.go` for the pattern).
- Handle `sql.ErrNoRows` explicitly.

### 6.5 Go style

- Go 1.24 across all components. `gofmt -s` compliant. `golangci-lint` clean.
- Structured logging with zerolog (backend/collect) or custom logger (sync). Automatic redaction of secrets/tokens/passwords.
- No comments that explain WHAT the code does — identifiers do that. Only comment non-obvious WHY.

### 6.6 Alerting conventions (this branch)

- **Never trust identity labels from clients**: `collect/methods/mimir.go:injectLabels` overwrites `system_*` and `organization_*` authoritatively. Empty DB values strip the label rather than leave the client's.
- **`organization_id` on `alert_history`** is the single source of truth for tenant scoping. Never filter by `system_key` alone.
- **`resolveOrgID` semantics** (`backend/methods/alerting.go`):
  - Customer → pinned to own org (query param ignored)
  - Distributor/Reseller → must pass `organization_id`, validated via hierarchy
  - Owner → `organization_id` optional; empty means "aggregate all tenants" (only meaningful for totals/trend)
  - Mimir-backed endpoints (`/alerts`, `/alerts/config`) reject empty via `requireOrgID`
- **Webhook receiver URLs** validated against SSRF ranges (loopback, RFC1918, link-local, IMDS, non-http(s)).
- **`system_key` in `SystemOverride`** must match `^[A-Za-z0-9_:.\-]+$` to avoid Alertmanager YAML matcher injection.

---

## 7. Testing

### 7.1 Test tokens

Generated by `make gen-tokens` in `backend/`. Files: `backend/token-owner`, `token-distributor`, `token-reseller`, `token-customer`. Use with:

```bash
curl -H "Authorization: Bearer $(cat backend/token-owner)" http://localhost:8080/api/alerts/totals
```

### 7.2 Running tests

```bash
cd <component> && make test            # all tests
make test-coverage                     # coverage.html
go test -race ./...                    # race detection
go test -v ./<package>                 # focused
```

Backend integration tests use `testutils/` for mock users/tokens/JWT. SQL mocks via `github.com/DATA-DOG/go-sqlmock`.

---

## 8. Development Workflow

### 8.1 First-time setup

```bash
cd <component> && make dev-setup   # downloads deps, copies .env.example
make dev-up                         # starts postgres + redis
cd backend && make db-migrate
```

### 8.2 Running services

```bash
cd backend && make run             # :8080
cd collect && make run             # :8081
cd frontend && npm run dev         # :5173
make run-qa                        # uses .env.qa (backend and collect)
```

### 8.3 Build & release

```bash
make build        # single platform -> build/
make build-all    # linux/darwin/windows × amd64/arm64
./release.sh patch|minor|major    # bumps version.json across all components
```

---

## 9. Database

- **Master schema**: `backend/database/schema.sql` — complete current structure, used by fresh deployments.
- **Migrations**: `backend/database/migrations/NNN_description.sql` + `NNN_description_rollback.sql`. Latest: **019_add_alert_history** (includes `organization_id` column for tenant scoping).
- Any DB change MUST update BOTH the migration AND `schema.sql`. Fresh deployments use `schema.sql`; upgrades apply migrations.
- Models under `backend/models/` and affected SQL queries must be updated alongside.
- OpenAPI spec must be updated if response shapes change.

---

## 10. API

### 10.1 Adding an endpoint

1. Route in `backend/main.go` with middleware chain.
2. Handler in `backend/methods/` (service/entity calls).
3. Request/response models in `backend/models/`.
4. **Update `backend/openapi.yaml`** (mandatory — validated by `make validate-docs`).
5. `make pre-commit`.

### 10.2 API reference

Authoritative: `backend/openapi.yaml` (also `make docs` / redocly). High-level route groups in `backend/main.go`:

```
/api/health                         public
/api/auth/*                         exchange/refresh public, rest JWT-protected
/api/me, /api/me/*                  self-service profile + impersonation
/api/distributors|resellers|customers/*    CRUD + import/export + totals/trend
/api/users/*                        CRUD + avatar + import/export + password reset + suspend/reactivate
/api/systems/*                      CRUD + inventory + alerts + regenerate-secret + reachability + export
/api/applications/*                 CRUD + assign/unassign org + totals/summary/trend
/api/alerts, /api/alerts/{totals,trend,config}   active alerts + config + aggregates
/api/filters/{systems,applications,users}  UI filter aggregation
/api/rebranding/*                   per-org per-product asset management
/api/organizations, /api/roles, /api/organization-roles  metadata
/api/validators/vat/:entity_type    VAT validation
/api/stats                          Owner-only platform stats

# collect (:8081)
POST /api/systems/inventory                   HTTP Basic
POST /api/systems/heartbeat                   HTTP Basic
ANY  /api/services/mimir/alertmanager/api/v2/{alerts,silences}[/*]   HTTP Basic (Mimir proxy)
POST /api/alert_history                       Bearer (Alertmanager webhook)
```

If a route exists in `main.go` but isn't in `openapi.yaml`, that's the bug — fix `openapi.yaml`.

---

## 11. Component-specific pointers

Operational runbooks and command cookbooks live with each component:

- `backend/README.md` — backend-specific env, tooling, migrations.
- `collect/README.md` — worker model, differ config, queue ops.
- `sync/README.md` — init/sync/pull/prune workflows, Logto setup.
- `services/mimir/README.md` — Mimir single-node setup, scripts.
- `frontend/README.md` — Vue/Vite specifics.

When implementing a feature, prefer consulting the component README over re-deriving from CLAUDE.md.

---

## 12. Common pitfalls

- **Case-sensitivity on org_role**: JWT carries "Owner"/"Distributor"/etc.; middleware/jwt.go normalizes to lowercase. Always switch on lowercase values.
- **`system_key` hidden for unregistered systems**: `GetSystem` blanks `SystemKey` when `RegisteredAt IS NULL`. Per-system alert endpoints return empty for unregistered systems — this is by design, not a bug.
- **`X-Scope-OrgID` never from the request**: derived server-side from DB (collect) or JWT (backend). Treat any PR that lets it be set by the client as a security defect.
- **Pre-commit OpenAPI validation**: uses `redocly` CLI. If missing, install via npm or the validate-docs target will fail silently in CI.
- **`podman` vs `docker`**: local dev uses `podman`; Makefiles handle this but ad-hoc commands in docs examples may say `docker` — substitute accordingly.
