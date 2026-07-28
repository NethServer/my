# Threat Shield — Distributed Probe Threat Feed

**Status:** approved design
**Date:** 2026-07-28
**Repos touched:** `my`, `nethsecurity`, `ns8-crowdsec`, `ns8-loki`
**Official language:** English. An Italian summary exists at `2026-07-28-threat-shield-riassunto-it.md` and is a translation, not a source of truth.
**Originating proposal:** inlined as Appendix A. This document supersedes it; where the two
disagree, this document wins and §A.4 records why.

## 1. Goal and scope

Turn the NethSecurity and NethServer 8 fleet into a network of security probes whose
observations become a product: a high-confidence IP blocklist distributed from an
authenticated endpoint on `my`.

The originating proposal (Appendix A) describes five products at once (health score,
configuration drift, capacity sizing, probe sensing, blocklist API). That is too much for
one design, so the work is decomposed:

| # | Sub-project | Depends on | Covered here |
|---|---|---|---|
| 0 | Probe signal ingest contract (versioned envelope, sanitizer, event store) | — | yes |
| 1 | Threat Shield (consensus → authenticated IP feed) | 0 | yes |
| 2 | Fleet metrics ingest (P85 capacity data) | 0 | no |
| 3 | Health score and configuration drift | 2 + inventory | no |
| 4 | Capacity sizing advisory | 2 | no |

Sub-projects 0 and 1 are specified together because 1 is the first consumer of 0 and
proves the contract. Sub-projects 2–4 each get their own spec.

### 1.1 Non-goals

- No fleet-wide metrics pipeline. Neither client remote-writes metrics today, and Threat
  Shield does not need time series.
- No TimescaleDB. See §4.1.
- No replacement for CrowdSec CAPI. See §1.2.
- No fleet-wide threat dashboard in v1. See §9.

### 1.2 Relationship to CrowdSec CAPI

`ns8-crowdsec` already shares signals with, and consumes blocklists from, CrowdSec's
central API. Threat Shield overlaps it. The differentiators are narrow and should be
stated honestly rather than oversold:

- NethSecurity firewall signal (WAN drops, banip hits) and NethVoice SIP signal, which
  CAPI never sees.
- Provenance visible to the tenant inside `my`: an org can see what its own probes
  reported and why an IP is listed.
- Works with CAPI disabled, which some customers require.

Threat Shield is not presented as strictly better than CAPI. Both can run.

## 2. Decisions

| # | Decision | Rationale |
|---|---|---|
| D1 | Feed fetched with appliance credentials (`system_key:system_secret`, HTTP Basic); content is a single global list, identical for every subscriber | Reuses the auth `collect` already has for inventory/heartbeat. No key handling for humans, no per-tier feed generation. |
| D2 | Reporting via telegraf `outputs.http` on NethSecurity; a Python forwarder service on NS8 | Both agents are already installed and sanctioned. No new daemon on NethSecurity. |
| D3 | v1 categories: `port_scan`, `ssh_bruteforce`, `http_exploit`, `sip_probe` | Availability differs per product (§5.1); the ingest contract is category-generic so adding one is config, not schema. |
| D4 | Participation on by default, per-organization opt-out. No reciprocity gate | Fastest path to probe critical mass. Legal basis is legitimate interest (GDPR Art. 6(1)(f)), which Recital 49 names explicitly for network and information security — not consent. Documented opt-out plus §3 sanitization is the compensating control. |
| D5 | Promotion rule: ≥3 distinct systems across ≥2 distinct organizations within a rolling 60 min. Entry expires 24 h after last report; new reports refresh the TTL | The cross-org requirement removes the single-misconfigured-fleet failure mode. Short TTL prevents rented and NAT addresses from lingering after reassignment. |
| D6 | Plain partitioned Postgres on the existing instance; no new datastore | Consensus is a 60-minute `GROUP BY` over ≤7 days of rows. §4.1. |
| D7 | Ingest and feed live on `collect`; read and management APIs on `backend` | `collect` owns appliance Basic auth, Redis queues, workers and cron. `backend` owns the JWT/RBAC plane. Same split as inventory, heartbeat and backups. |
| D8 | NS8 reporter lives in `ns8-crowdsec` (primary) and `ns8-loki` (fallback) | CrowdSec alerts carry the attacker IP; the Prometheus series `ns8-metrics` scrapes are aggregates that have discarded it. |
| D9 | Feed ships dark; published only after an adoption floor is cleared | §8. |

## 3. Architecture

```
+-- NethSecurity probe --------------+  +-- NS8 probe -----------------------+
| telegraf                           |  | PRIMARY: ns8-crowdsec              |
|  inputs.tail  nft drops, banip hits|  |  threat-shield-forwarder.service   |
|  inputs.tail  sshd + ns-api auth   |  |  cscli alerts list -o json         |
|  inputs.tail  nginx probe URIs     |  |  local-origin only (no CAPI)       |
|  processors   dedup, scrub, aggr.  |  |  scenario -> category map          |
|  outputs.http --> my               |  | FALLBACK: ns8-loki (no crowdsec)   |
|  creds: ns-plug system_key/secret  |  |  LogQL {category="security"}       |
+--------------+---------------------+  |  creds + lifecycle from            |
               |                        |  subscription-changed event        |
               |                        +---------------+--------------------+
               +--- HTTPS Basic (system_key:system_secret) ---+
                                  v
+-- collect (:8081) ----------------------------------------------------------+
| POST /api/systems/threat-events   Basic auth (existing middleware)          |
|   validate -> sanitize -> drop non-public IPs -> Redis queue                |
| ThreatEventWorker   batch INSERT (workers/ pattern)                         |
| ConsensusCron 5min  promote / expire / materialize (cron/ pattern, SET NX)  |
| GET /api/systems/threat-shield/blocklist  Basic auth, ETag/304, gzip        |
+--------------+--------------------------------------------------------------+
               v
  Postgres (existing)                        Redis (existing)
  threat_events      partitioned daily, 7d    ts:blocklist:v1  body/gz/etag
  threat_blocklist   current feed, durable
  threat_allowlist   owner-managed excludes
  threat_daily_stats long-term rollup
               v
  backend (:8080) JWT + RBAC — tenant/Owner read APIs, allowlist CRUD,
                               participation toggle
```

## 4. Data model

Migration `backend/database/migrations/039_add_threat_shield.sql` plus
`039_add_threat_shield_rollback.sql`, and `backend/database/schema.sql` updated in the
same change (both are mandatory per `AGENTS.md` §9).

```sql
CREATE TABLE threat_events (
    observed_at      TIMESTAMPTZ  NOT NULL,
    id               BIGSERIAL    NOT NULL,
    system_id        VARCHAR(255) NOT NULL,
    organization_id  VARCHAR(255) NOT NULL,
    attacker_ip      INET         NOT NULL,
    category         VARCHAR(32)  NOT NULL,
    target_port      INTEGER,
    hit_count        INTEGER      NOT NULL DEFAULT 1,
    metadata         JSONB,
    PRIMARY KEY (observed_at, id)
) PARTITION BY RANGE (observed_at);          -- daily partitions, 7 day retention

-- retry idempotency: reporter redelivery must not inflate hit_count
CREATE UNIQUE INDEX idx_threat_events_dedup
    ON threat_events (system_id, attacker_ip, category, observed_at);
CREATE INDEX idx_threat_events_ip   ON threat_events (attacker_ip, observed_at DESC);
CREATE INDEX idx_threat_events_org  ON threat_events (organization_id, observed_at DESC);

CREATE TABLE threat_blocklist (
    attacker_ip      INET PRIMARY KEY,
    first_listed_at  TIMESTAMPTZ NOT NULL,
    last_seen_at     TIMESTAMPTZ NOT NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    distinct_systems INTEGER NOT NULL,
    distinct_orgs    INTEGER NOT NULL,
    categories       TEXT[]  NOT NULL,
    listing_reason   JSONB   NOT NULL   -- promote-time evidence, outlives the partitions
);

CREATE TABLE threat_allowlist (
    cidr        CIDR PRIMARY KEY,
    reason      TEXT NOT NULL,
    created_by  VARCHAR(255) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ                       -- NULL = permanent
);

CREATE TABLE threat_daily_stats (                 -- survives partition drop
    day           DATE NOT NULL,
    category      VARCHAR(32) NOT NULL,
    distinct_ips  INTEGER NOT NULL,
    total_hits    BIGINT  NOT NULL,
    PRIMARY KEY (day, category)
);

CREATE TABLE threat_shield_participation (
    organization_id VARCHAR(255) PRIMARY KEY,
    enabled         BOOLEAN NOT NULL,
    updated_by      VARCHAR(255) NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Three choices worth defending:

**`listing_reason`** snapshots promotion evidence at decision time. Raw events live seven
days; without the snapshot, "why is this IP listed?" is unanswerable a week later — and
that question gets asked by whoever's customer just got blocked.

**`threat_daily_stats`** is the long-term analytics asset. Rolling up before partitions
drop is what turns a blocklist into fleet threat-trend data, at a few rows per day.

**Absent participation row means participating**, which encodes on-by-default without
backfilling every organization. `collect` caches the opt-out set in Redis with a short
TTL so ingest does not hit Postgres per request.

### 4.1 Why not TimescaleDB

The originating proposal (Appendix A §A.2) specifies hypertables and continuous
aggregates. Threat Shield does not
benefit: consensus is one 60-minute aggregation over at most seven days of rows, which
native daily partitions serve comfortably. Adopting the extension now would add a
dependency to a managed Postgres plus compression and retention policies to operate, for
no gain. Its real justification is sub-project 2 (30-day P85 rollups across the fleet), so
the decision belongs there, made on measured cost. Nothing here blocks that move:
`threat_events` is append-only and partitioned.

## 5. Ingest contract

`POST /api/systems/threat-events` on `collect`, HTTP Basic with the system's
`system_key:system_secret`.

```jsonc
{
  "probe":  { "product": "nethsecurity", "version": "8.7.2", "reporter": "telegraf-1.30" },
  "window": { "start": "2026-07-28T10:00:00Z", "end": "2026-07-28T10:05:00Z" },
  "threat_events": [
    { "attacker_ip": "198.51.100.44", "category": "ssh_bruteforce",
      "target_port": 22, "hit_count": 85, "metadata": { "distinct_ports": 1 } }
  ]
}
```

The envelope is versioned and category-generic on purpose: sub-projects 2–4 add a sibling
array (for example `capacity_metrics`), not a new endpoint or auth path.

**Server-authoritative identity.** `system_id` and `organization_id` are resolved from the
authenticated system row and never read from the body. A payload that carries them is
rejected rather than silently overridden — a client sending them signals a broken
reporter. This mirrors the rule already enforced by `collect/methods/mimir.go:injectLabels`
and by `alert_history`.

### 5.1 Category availability

| category | NethSecurity | NS8 (crowdsec) | NS8 (loki fallback) |
|---|---|---|---|
| `port_scan` | nft drops + banip | — | — |
| `ssh_bruteforce` | sshd, ns-api | `crowdsecurity/ssh-bf` | sshd lines |
| `http_exploit` | nginx | `crowdsecurity/http-probing` | traefik streams |
| `sip_probe` | — | asterisk scenarios (`ns8-nethvoice`) | asterisk streams |

Availability is asymmetric and the design states it rather than pretending parity. SIP is
NethServer 8 only; NethSecurity's port-5060 hits fold into `port_scan`.

### 5.2 Sanitizer

`collect/services/threatsanitize/`, applied before the Redis queue:

- `attacker_ip` must parse as a public unicast address. RFC1918, loopback, CGNAT
  (100.64/10), link-local, IMDS (`169.254.169.254`), multicast, IPv6 ULA and the
  reporter's own egress IP are dropped. This is a hard filter **at ingest**, not only at
  promotion, so private addressing never reaches the store.
- `metadata` is filtered against a per-category key allowlist with typed values. Unknown
  keys are dropped; nothing free-text is stored.
- Category-specific redaction: usernames are dropped entirely, never hashed (a hash of
  `root` or `admin` is trivially reversible and carries no analytic value); URIs are
  reduced to the matched signature from a fixed pattern list (`/.env`, `/wp-login.php`),
  never the raw request line; SIP extensions are replaced by a length bucket.
- Caps: 500 events per request, 4 KB metadata per event, per-system rate limit reusing the
  backups limiter. Over-cap requests are **truncated with a counter, not rejected** — a
  probe under active attack must not lose its whole batch.

## 6. Consensus and feed

`collect/cron/threat_consensus.go`, every 5 minutes, guarded by a Redis `SET NX` lock (the
pattern `collect` already uses for backup retention) so two instances cannot both
generate.

```sql
-- promote
INSERT INTO threat_blocklist AS b (attacker_ip, first_listed_at, last_seen_at, expires_at,
                                   distinct_systems, distinct_orgs, categories, listing_reason)
SELECT e.attacker_ip, NOW(), MAX(e.observed_at), MAX(e.observed_at) + INTERVAL '24 hours',
       COUNT(DISTINCT e.system_id), COUNT(DISTINCT e.organization_id),
       ARRAY_AGG(DISTINCT e.category),
       jsonb_build_object('systems', COUNT(DISTINCT e.system_id),
                          'orgs',    COUNT(DISTINCT e.organization_id),
                          'hits',    SUM(e.hit_count),
                          'window',  '60m', 'rule', 'v1')
FROM threat_events e
WHERE e.observed_at >= NOW() - INTERVAL '1 hour'
  AND NOT EXISTS (SELECT 1 FROM threat_allowlist a
                  WHERE e.attacker_ip <<= a.cidr
                    AND (a.expires_at IS NULL OR a.expires_at > NOW()))
GROUP BY e.attacker_ip
HAVING COUNT(DISTINCT e.system_id) >= 3 AND COUNT(DISTINCT e.organization_id) >= 2
ON CONFLICT (attacker_ip) DO UPDATE
  SET last_seen_at     = GREATEST(b.last_seen_at, EXCLUDED.last_seen_at),
      expires_at       = EXCLUDED.expires_at,      -- TTL refreshed by new hits
      distinct_systems = EXCLUDED.distinct_systems,
      distinct_orgs    = EXCLUDED.distinct_orgs,
      categories       = EXCLUDED.categories;
      -- first_listed_at deliberately untouched

DELETE FROM threat_blocklist WHERE expires_at <= NOW();
```

The allowlist is applied **at promotion, not at read**, so adding an entry retroactively
unlists on the next 5-minute pass. Fleet egress IPs join the same `NOT EXISTS` through a
maintained CIDR set.

**Fleet self-protection.** `collect` records each system's observed source IP at ingest;
the union of current fleet egress IPs is an automatic exclusion set. This closes the worst
failure mode — one customer's misconfigured appliance getting the whole fleet's own WAN
address listed — without depending on inventory contents.

**Materialization.** After each pass the sorted list is rendered once into Redis:
`ts:blocklist:v1:body`, `:gz`, `:etag` (SHA-256 of body), `:generated_at`. Serving never
touches Postgres, so feed cost is flat regardless of subscriber count and a database
hiccup cannot blank the list.

**Feed response** — `GET /api/systems/threat-shield/blocklist`, Basic auth:

```
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
ETag: "sha256-9f2b..."
Cache-Control: max-age=900

# nethesis threat shield v1
# generated: 2026-07-28T10:05:00Z  entries: 1843  rule: 3 systems / 2 orgs / 60m / 24h ttl
198.51.100.44
203.0.113.12
```

`If-None-Match` returns `304`, which is the normal answer to most polls at a 15-minute
regeneration cadence. Plain text serves both consumers with no per-client format branch.
Hard cap 50 000 entries, deterministic sort order.

Participation is decoupled from consumption per D4: an organization that opts out stops
reporting and keeps receiving the feed.

## 7. Client integration

### 7.1 NethSecurity (`nethsecurity`)

Telegraf is already installed and is the sanctioned metrics path, so this is configuration
plus a scrubbing processor chain, not a new daemon.

```toml
# inputs: nft drops / banip hits, sshd + ns-api auth failures, nginx probe URIs
[[inputs.tail]]
  # ...
[[outputs.http]]
  url          = "https://my.nethesis.it/api/systems/threat-events"
  username     = "$SYSTEM_KEY"
  password     = "$SYSTEM_SECRET"
  data_format  = "json"
```

Feed consumption reuses the existing product feature also called Threat Shield (`banip`
behind it), so this is a new Nethesis-managed blocklist entry rather than new UI:
`ns-plug` fetches with subscription credentials and an ETag cache, writes
`/etc/banip/nethesis-threat-shield.txt`, and banip consumes a local file. Credentials stay
out of banip's configuration and a fetch failure degrades to the last good file.

### 7.2 NethServer 8 primary (`ns8-crowdsec`)

A `threat-shield-forwarder.service` plus timer, modeled on
`ns8-loki/imageroot/bin/cloud-log-manager-forwarder` (Python systemd service, watermark
file, batched JSON POST, lifecycle driven by the `subscription-changed` event).

Source is `cscli alerts list --output json`, mapping scenario to category.

**Local-origin only.** `ns8-crowdsec/imageroot/actions/list-detections/10list-detections`
already avoids `--all` because that pulls CAPI-origin community alerts. The forwarder must
apply the same rule: re-reporting CrowdSec's community list into our consensus would
manufacture agreement across organizations and poison the feed.

Feed consumption imports the list as decisions, reusing the already-registered firewall
bouncer with no new enforcement path:

```
cscli decisions import -i threat-shield.txt --format values \
      --duration 24h --scenario nethesis/threat-shield
```

The dedicated scenario name keeps Nethesis-sourced bans distinguishable from local and
CAPI decisions in `cscli decisions list`, so an admin can always see why an IP is banned,
and `cscli decisions delete --scenario nethesis/threat-shield` is a clean escape hatch.

### 7.3 NethServer 8 fallback (`ns8-loki`)

For clusters without CrowdSec: the same forwarder shape querying Loki with LogQL over
`{category="security"}` streams. Shared code with 7.2 is the POST-plus-watermark helper
(about 40 lines); the query sides are genuinely different, so no shared abstraction is
introduced.

## 8. Cold start

The promotion rule means the feed is empty until enough probes report, so the feed ships
**dark**: phases 1–2 collect and promote normally while `GET /blocklist` stays behind a
configuration flag until the fleet clears a floor — proposed ≥50 reporting systems across
≥10 organizations, plus one week of statistics reviewed as sane. Publishing an empty or
five-entry list to production firewalls would burn the feature's credibility on day one
and teach administrators to ignore it.

## 9. `my` APIs, RBAC and UI

New Logto resource `threat_shield` (`read:threat_shield`, `manage:threat_shield`), which
means `sync/configs/config.yml` plus a `sync` run, not only backend code. Owner and Admin
get manage, Support gets read, Customer reads its own data only.

```
GET    /api/threat-shield/stats              read    own contribution; Owner: fleet-wide
GET    /api/threat-shield/events             read    own reported events (AppendOrgFilter)
GET    /api/threat-shield/participation      read    effective opt-in state
PUT    /api/threat-shield/participation      manage  opt out / opt back in
GET    /api/threat-shield/blocklist/:ip      manage  Owner: listing_reason provenance
GET    /api/threat-shield/allowlist          manage  Owner
POST   /api/threat-shield/allowlist          manage  Owner
DELETE /api/threat-shield/allowlist/:cidr    manage  Owner
```

`resolveOrgID` semantics are copied from alerting: Customer pinned to its own organization,
Distributor and Reseller must pass a hierarchy-validated `organization_id`, Owner may omit
it to aggregate. `backend/openapi.yaml` is updated in the same change — `make
validate-docs` enforces it.

Frontend v1 is deliberately minimal: participation toggle in organization settings, a
threat summary card (own reported events, categories, 7-day trend), and Owner-only
allowlist CRUD with provenance lookup. The fleet-wide threat dashboard is deferred until
sub-projects 2–4 make it worth building once.

## 10. Error handling

- A client must **never write an empty list on error**. An empty file means "no threats",
  which silently disables protection. Only a `200` with a well-formed body (header line
  present, at least one entry, sane size) replaces the local file; failures keep the
  previous file and log.
- Reporter watermarks advance only after a `2xx`, so a `my` outage produces delayed events
  rather than lost ones. Both reporters cap catch-up depth to avoid a flood after a long
  outage.
- Ingest is fail-closed on authentication and fail-open on content: a malformed event is
  dropped with a counter, the rest of the batch is accepted.
- If consensus generation fails, the previous Redis snapshot keeps being served with its
  original `generated_at`. Subscribers get a stale list, never a blank one, and the header
  timestamp lets a client decide to distrust an old feed.

## 11. Testing

- **Sanitizer** — table-driven over every IP class (RFC1918, CGNAT, IMDS, multicast, IPv6
  ULA, fleet egress), metadata allowlist drops, username stripping, URI signature
  reduction. Highest-value surface here: a sanitizer bug is a data-protection incident.
- **Worker** — idempotent redelivery of the same batch must not inflate `hit_count`,
  asserting the unique index actually holds.
- **Consensus** — integration test against the dev Postgres from `make dev-up`, with
  fixtures on the boundary: 3 systems in 1 org must not promote, 3 systems across 2 orgs
  must, an allowlisted IP must not, expiry and TTL refresh both asserted. `go-sqlmock`
  cannot validate this, because the logic *is* the SQL.
- **Feed** — ETag/304, gzip, 50 000 cap, empty-body guard.
- **End-to-end** via `apitool` — two organizations, three registered systems, real Basic
  auth; assert promotion end to end plus the 401/403 paths, and that opt-out stops ingest
  while the feed still serves.
- **Clients** — `telegraf --test` on the NethSecurity configuration; robot tests in
  `ns8-crowdsec/tests/` and `ns8-loki/tests/` following each module's existing
  `10__check_services.robot` pattern.

## 12. Implementation plan

Phase 1 is this spec's implementation plan. Phases 2 and 3 are tracked separately: client
work is specified here as a contract (envelope, auth, categories, metadata allowlist) and
lands as one PR per repo, since each has its own review, release and image-build cycle —
folding them into one plan would stall all of them behind the slowest.

### Phase 1 — `my` (ordered)

1. Migration `039_add_threat_shield.sql` + rollback + `schema.sql`, including the daily
   partition helper and 7-day drop.
2. `collect/services/threatsanitize/` with its test table. Written first because
   everything downstream depends on its output shape.
3. `collect` models plus `POST /api/systems/threat-events`: Basic auth, envelope
   validation, authoritative identity, caps, Redis enqueue.
4. `ThreatEventWorker` batch insert with idempotent redelivery, registered in
   `workers/manager.go`.
5. Participation opt-out check with Redis-cached opt-out set, enforced at ingest.
6. `collect/cron/threat_consensus.go`: promote, expire, roll up
   `threat_daily_stats`, materialize the Redis snapshot, all under `SET NX`.
7. `GET /api/systems/threat-shield/blocklist`: Basic auth, ETag/304, gzip, cap,
   dark-launch flag.
8. Partition maintenance and retention in the same cron.
9. RBAC: `threat_shield` resource in `sync/configs/config.yml`, `sync` run.
10. `backend` read APIs of §9 with `resolveOrgID`, plus allowlist CRUD and participation
    toggle; `openapi.yaml` updated.
11. `apitool` end-to-end scenario.
12. `make pre-commit` in `collect`, `backend`, `sync`.

### Phase 2 — probes (parallel, one PR per repo)

- `ns8-crowdsec`: forwarder service and timer, scenario→category map, local-origin filter,
  feed importer.
- `ns8-loki`: fallback forwarder over `{category="security"}`.
- `nethsecurity`: telegraf inputs, scrub processors, http output; `ns-plug` feed fetcher
  into banip.

### Phase 3 — `my`

Frontend of §9, statistics views, documentation, and un-darkening the feed once the §8
floor is cleared.

## 13. What this leaves for sub-projects 2–4

The versioned `probe` envelope on an existing authenticated endpoint, per-system source-IP
tracking, the daily-rollup pattern that lets raw data expire without losing history, the
participation table, and the `threat_shield` RBAC resource are all reusable. The
TimescaleDB question stays open and is decided in sub-project 2 on measured rollup cost.

Appendix A §A.2–§A.3 hold the proposal's material for those sub-projects — the
`fleet_telemetry` column set, the health-score and sizing rules, and the frontend surfaces.
It is inlined here so those specs start from the recorded intent rather than from memory,
but none of it is approved by this document.

---

## Appendix A — originating proposal

Verbatim intent of the pre-design proposal (`data_value.md`, never committed), inlined so
this spec is self-contained. It arrived as a GitHub Spec-Kit package for
`.specify/specs/001-telemetry-fleet-insights/`; the packaging is dropped and only the
substance is kept. **Nothing in this appendix is a decision.** §A.4 maps each part to
where it landed.

### A.1 Business intent and user stories

Turn opt-in heartbeat, performance metrics and edge threat signals from NS8 and
NethSecurity nodes into operational tools inside `my`, across five dimensions:

1. **Node health score** — algorithmic 0–100% index from resource pressure and service
   states.
2. **Fleet-wide configuration drift alerts** — proactive detection of non-standard,
   vulnerable or unencrypted configurations.
3. **Capacity planning and hardware sizing advisory** — predictive sizing from 30-day
   *P85* resource saturation trends.
4. **Distributed sensor and threat probe metrics** — network, app-layer and host-level
   probe signals (scanning, auth velocity, SIP probing, process anomalies) to spot
   emerging attack patterns.
5. **Dynamic IP blocklist API** — high-confidence global feed from multi-node consensus.

**Story 1 — distributed threat and anomaly probe sensing.** As a security analyst, I want
nodes acting as edge sensors logging connection spikes, failed logins, suspicious URI
requests and SIP probing, so `my` can identify exploit trends before vendor CVEs.
Acceptance: nodes send lightweight summaries of port scans, failed-auth velocity,
suspicious HTTP User-Agents/URIs and SIP `INVITE` bursts; process anomalies (unexpected
shell execution under `www-data` or `asterisk`) raise high-priority threat events;
sensitive parameters (usernames, local IPs) are scrubbed before transmission.

**Story 2 — node health score and capacity sizing.** As an MSP administrator, I want a
0–100% health score and predictive hardware upgrade warnings per node. Acceptance: score
drops on RAM pressure (>90%), IO wait (>15%) or critical drift; the sizing engine advises
when 30-day P85 connection/user thresholds are crossed.

**Story 3 — multi-node consensus blocklist.** As an MSP or enterprise security subscriber,
I want an API on `my` returning malicious IPs seen across the global sensor network.
Acceptance: an IP needs ≥3 distinct nodes within a rolling 60-minute window to enter the
public feed; the endpoint requires subscription-token auth and returns plain-text IP/CIDR.

### A.2 Proposed architecture and schema

Pipeline: edge nodes (system/app metrics + threat probe signals) → HTTPS heartbeat with
Bearer token → `my` backend (ingestion + Bearer auth middleware, PII/network sanitizer,
dispatcher) → PostgreSQL with **TimescaleDB** (hypertables `fleet_telemetry` and
`probe_threat_events`; continuous aggregates for hourly P85 metrics and 1-hour threat
consensus) → advisory engine and API service (health score and sizing rules, threat feed
at `/api/v1/threat-shield/blocklist`).

```sql
CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE fleet_telemetry (
    recorded_at           TIMESTAMPTZ NOT NULL,
    org_id                UUID NOT NULL,
    node_id               UUID NOT NULL,
    cpu_usage_pct         REAL,
    ram_used_bytes        BIGINT,
    ram_total_bytes       BIGINT,
    io_wait_pct           REAL,
    conntrack_active      INTEGER,
    active_lan_devices    INTEGER,
    mail_active_users     INTEGER,
    voip_concurrent_calls INTEGER
);
SELECT create_hypertable('fleet_telemetry', 'recorded_at');
ALTER TABLE fleet_telemetry SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'org_id, node_id',
    timescaledb.compress_orderby   = 'recorded_at DESC'
);
SELECT add_compression_policy('fleet_telemetry', INTERVAL '7 days');

CREATE TABLE probe_threat_events (
    recorded_at         TIMESTAMPTZ NOT NULL,
    reporting_node_hash TEXT        NOT NULL,
    attacker_ip         INET        NOT NULL,
    attack_category     VARCHAR(32) NOT NULL, -- ssh_bruteforce, port_scan, sip_probe, http_exploit
    target_port         INTEGER,
    event_metadata      JSONB                 -- sanitized headers, URI signatures, shell alerts
);
SELECT create_hypertable('probe_threat_events', 'recorded_at');
CREATE INDEX idx_probe_threats_consensus
    ON probe_threat_events (recorded_at DESC, attacker_ip);
```

Proposed ingest payload, `POST /api/v1/node/telemetry` — one endpoint carrying metrics and
threat events together:

```json
{
  "timestamp": 1779972600,
  "system": {
    "cpu_usage_pct": 18.2,
    "ram_used_bytes": 5153960755,
    "ram_total_bytes": 8589934592,
    "io_wait_pct": 0.8
  },
  "probe_security_events": [
    { "attacker_ip": "198.51.100.44", "category": "ssh_bruteforce", "target_port": 22,
      "metadata": { "failed_attempts_1h": 85 } },
    { "attacker_ip": "203.0.113.12", "category": "sip_probe", "target_port": 5060,
      "metadata": { "method": "INVITE", "target_extension": "90011234" } },
    { "attacker_ip": "192.0.2.88", "category": "http_exploit", "target_port": 443,
      "metadata": { "uri_pattern": "/.env", "user_agent": "Go-http-client/1.1" } }
  ]
}
```

Proposed consensus query:

```sql
SELECT attacker_ip,
       COUNT(DISTINCT reporting_node_hash) AS consensus_score,
       ARRAY_AGG(DISTINCT attack_category) AS attack_types
FROM probe_threat_events
WHERE recorded_at >= NOW() - INTERVAL '1 hour'
GROUP BY attacker_ip
HAVING COUNT(DISTINCT reporting_node_hash) >= 3
ORDER BY consensus_score DESC
LIMIT 50000;
```

### A.3 Proposed task list

*Phase 1 — database and ingestion:* enable `timescaledb`; run the hypertable migration;
create `/api/v1/node/telemetry` with node Bearer auth; build the PII/network sanitizer
that hashes node identifiers and scrubs raw parameters.

*Phase 2 — collectors and analytics:* lightweight bash/python probe collector on
NS8/NethSecurity gathering port scans, auth failures and SIP probing; hourly health-score
worker over RAM, IO wait and drift penalties; sizing engine on
`percentile_cont(0.85)` to flag under-provisioned nodes; 1-hour rolling consensus job
generating the blocklist.

*Phase 3 — frontend and API:* health-score badge plus breakdown modal on node views; a
"fleet configuration drift and threat summary" tab on the MSP organization dashboard; a
capacity-planning advisory widget under node settings; the blocklist endpoint with API-key
authorization.

### A.4 Where each part landed

| Proposal element | Disposition |
|---|---|
| Five products in one spec | Split into sub-projects 0–4 (§1). Only 0 and 1 are designed here. |
| TimescaleDB hypertables, compression, continuous aggregates | Rejected for v1; native daily partitions instead (§4.1). Reconsidered in sub-project 2 on measured cost. |
| `probe_threat_events` with `reporting_node_hash` | Became `threat_events` (§4) with real `system_id` + `organization_id`, needed for the cross-org rule (D5) and for tenant provenance. Hashing the node would make both impossible. |
| `fleet_telemetry` hypertable | Deferred to sub-project 2. Threat Shield needs no time series (§1.1). |
| Bearer token auth, `/api/v1/...` paths | Replaced by appliance Basic auth on `collect` paths (D1, D7) — reuses what `collect` already has, no new key handling. |
| One endpoint for metrics + threat events | Replaced by the versioned category-generic envelope (§5); sub-projects 2–4 add a sibling array, not a new endpoint. |
| Consensus: ≥3 nodes / 60 min | Tightened to ≥3 systems across ≥2 organizations, 24 h TTL refreshed by new hits (D5). Single-org agreement is not evidence. |
| Allowlist / fleet self-protection | Added, absent from the proposal (§6). |
| Sanitizer "hash node identifiers, scrub parameters" | Specified concretely (§5.2): hard public-IP filter at ingest, per-category metadata allowlist, usernames dropped rather than hashed, URI signature reduction, SIP extension bucketing. |
| Process-anomaly events (shell under `www-data`/`asterisk`) | Dropped from v1 categories (D3). Neither client emits them today; host-level process telemetry is a larger privacy conversation. |
| `user_agent` in metadata | Not in the allowlist. Free-text and fingerprintable. |
| Subscription-token / API-key authorized feed | Appliance credentials instead (D1); single global list, no per-tier generation. |
| Opt-in participation | On by default with per-org opt-out under legitimate interest (D4); consumption decoupled from participation. |
| New probe collector daemon on edge nodes | Rejected — telegraf on NethSecurity, `ns8-crowdsec`/`ns8-loki` forwarder on NS8 (D2, D8). |
| Health score, drift, sizing engine and their UI | Deferred to sub-projects 2–4 unchanged in intent; the thresholds in §A.1 are the recorded starting point. |
| Cold start | Absent from the proposal; feed ships dark behind an adoption floor (D9, §8). |
