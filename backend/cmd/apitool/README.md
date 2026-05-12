# apitool

Dev CLI that produces real, hierarchy-aware backend JWTs by running the full
Logto OIDC login + `/auth/exchange` flow (the same a browser does), and creates
test orgs, users, and systems autonomously. Replaces the old `gen-tokens` (which
locally signed JWTs whose embedded `org_id` had no counterpart in the database).

All credentials of generated users are persisted in `.api-registry.json` so any
later token request is a single command — no manual re-login.

## Why apitool

| Concern | `gen-tokens` (removed) | `apitool` |
|---|---|---|
| Auth method | Local JWT signing | Real OIDC login + `/auth/exchange` |
| `org_id` in token | Hardcoded fakes (`distributor-org-id`) | Real Logto IDs of existing orgs |
| RBAC hierarchy validation | Fails (org not in DB) | Works |
| Create orgs / users / systems | No | Yes |

## Setup

```bash
# From backend/
make apitool                    # builds ./apitool

./apitool init                  # one-time: prompts for OIDC config + owner
                                # creds, verifies via login + exchange, then
                                # saves to backend/.api-registry.json
                                # (mode 0600, gitignored)
```

`init` is interactive and asks for:

| Prompt | What it is | Example |
|---|---|---|
| Logto endpoint | Your Logto tenant URL | `https://your-tenant.logto.app` |
| Logto app ID | The OIDC app the backend uses | `abc123def456` |
| Auth base URL | Host serving `/login-redirect` | `https://my.example.com` |
| Backend URL | Backend API base (incl. `/api`) | `https://my.example.com/api` |
| Owner email | An existing Owner-role user | `owner@example.com` |
| Owner password | That user's password | — |

Nothing is baked in: re-run `init` to point at a different tenant or env. Press
Enter at any prompt to keep the previously-saved value (shown in `[brackets]`).

The owner credentials are stored in cleartext in the registry file; this is a
dev-only tool and the file never leaves your machine.

## Commands

```
apitool init
apitool token <name>                                # use "owner" for the owner
apitool list

apitool create-org <type> <name> --vat=<12 digits> [--description=...]
                                  [--data-<key>=<value>] [--as=<user-key>]
apitool delete-org <name>

apitool create-user --org=<name> --email=<email> --name='<name>'
                    [--role=Admin] [--username=...] [--key=<reg-name>] [--as=<user-key>]
apitool delete-user <key>
apitool cleanup-orphans --org=<name>                # purge users in org not in registry

apitool create-system --org=<customer-name> <system-name>  [--as=<user-key>]
```

`--as=<user-key>` runs the call authenticated as a registered user (default:
owner). Use it to build a real hierarchy where, for example, a Reseller is
created BY a Distributor user, so that Reseller becomes a child of that
Distributor in the RBAC graph.

`--data-<key>=<value>` on `create-org` is repeatable and populates arbitrary
fields under `custom_data` in the POST payload. Common keys: `address`,
`city`, `main_contact`, `email`, `phone`, `language` (`it`/`en`), `notes`.
Example:

```bash
./apitool create-org customer ACME --vat=123456789012 \
  --data-address='Via Roma 1' --data-city=Pesaro \
  --data-main_contact='Mario Rossi' --data-email=admin@acme.it \
  --data-language=it --data-notes='VIP'
```

`--role=` for `create-user` accepts the role name as exposed by `GET /api/roles`
(`Admin`, `Support`, `Backoffice`, `Reader`, `Super Admin`). Default `Admin`.

## Get a token, hit the API

```bash
TOK=$(./apitool token dist-admin)
curl -sk -H "Authorization: Bearer $TOK" https://my.example.com/api/me
```

Tokens are minted on every `apitool token` invocation by re-running the OIDC
flow with the saved credentials, so they're always fresh.

## Registry file

Path: `backend/.api-registry.json` (gitignored). Schema:

```json
{
  "config": { "logto_endpoint": "...", "logto_app_id": "...",
              "auth_base_url": "...", "backend_url": "..." },
  "owner":  { "email": "...", "password": "..." },
  "orgs":   { "<name>": { "type": "distributor|reseller|customer",
                          "logto_id": "...", "name": "...", "created_at": "..." } },
  "users":  { "<key>":  { "email": "...", "username": "...", "password": "...",
                          "logto_id": "...", "org_role": "...", "org_id": "...",
                          "org_name": "...", "created_at": "..." } }
}
```

`<name>` and `<key>` are the local aliases used by `--org=` and
`apitool token <key>`. Defaults: org name for orgs, email for users (override
with `--key`).

## Build a real test hierarchy

```bash
# Owner-level: top distributor
./apitool create-org distributor TestDist --vat=111111111111

# Distributor admin (so we can create things AS this distributor)
./apitool create-user --org=TestDist --email=dist@example.com --name='Dist Admin' \
                      --key=dist-admin

# Reseller created BY dist-admin → ends up under TestDist
./apitool create-org reseller TestRes --vat=222222222222 --as=dist-admin
./apitool create-user --org=TestRes --email=res@example.com --name='Res Admin' \
                      --key=res-admin

# Customers created BY res-admin → end up under TestRes
./apitool create-org customer TestCust1 --vat=333333333333 --as=res-admin
./apitool create-user --org=TestCust1 --email=c1@example.com --name='Cust1' \
                      --key=cust1-admin

# Systems live under customers
./apitool create-system --org=TestCust1 cust1-sys-A
```

Result: an Owner→Distributor→Reseller→Customer chain where every level has a
working user whose token reflects the real hierarchy.

## Push test alerts

`apitool` doesn't push alerts itself — they go straight to Mimir Alertmanager,
which is per-tenant by `X-Scope-OrgID`. The org's `logto_id` (visible via
`apitool list`) is the tenant ID:

```bash
ORG=$(./apitool list | awk '/TestCust1/ {print $3; exit}')
NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
END=$(date -u -v+1H +"%Y-%m-%dT%H:%M:%SZ")

curl -s -X POST "http://localhost:9009/alertmanager/api/v2/alerts" \
  -H "X-Scope-OrgID: $ORG" -H "Content-Type: application/json" \
  -d "[{\"startsAt\":\"$NOW\",\"endsAt\":\"$END\",
        \"labels\":{\"alertname\":\"HighCPU\",\"severity\":\"critical\",
                    \"system_key\":\"NETH-...\",\"instance\":\"test\"},
        \"annotations\":{\"summary\":\"Test alert\"}}]"
```

Then verify aggregation through `/api/alerts/totals` with each role's token.

## Known quirks

- **Customer ignores `organization_id`**: a Customer-role caller is always pinned
  to their own org. Passing a different `organization_id` does not return that
  org's data nor a `403` — it silently returns the caller's own data. Inherited
  from existing `resolveOrgID` logic; not specific to apitool.
- **Orphan users with `logto_id=None`**: if a `create-user` call partially fails
  (Logto sync errored mid-flow), the user can land in the local DB with
  `logto_id=null`. `cleanup-orphans` skips these because the API endpoints key
  off `logto_id`. Clean up via direct DB access if needed.

## Related

- [Backend README](../../README.md) — overall backend setup.
