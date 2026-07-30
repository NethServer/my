# Authorization regression suite

`my` is a maze of organization roles, technical roles, hierarchy scope and
credential types. This suite exists so that no change can quietly open a hole in
it. It provisions a real hierarchy through the public API, mints a real token per
persona via the full Logto OIDC flow, then fires every endpoint as every persona
and asserts what each one may and may not do.

```bash
cd backend
make apitool
./apitool authz provision     # create the fixture (idempotent)
./apitool authz personas      # verify tokens carry what config.yml promises
./apitool authz run           # all four layers → authz-report.json, exit 1 on failure
./apitool authz coverage      # every route has a declared intent?
./apitool authz teardown --yes
```

Useful flags: `--layer=gate|scope|apps|special`, `--filter=<substring>`,
`--personas=d1r1-admin,owner`, `--verbose`, `--idp-probe`, `--report=<path>`.

It refuses to run against anything but a local backend. It creates organizations
and users and fires destructive verbs, so pointing it at QA or production would
pollute real data. `--i-know-this-is-not-local` overrides, deliberately verbose.

## The one rule that keeps this honest

**Expectations are hand-written from intent, never read off the code.**

`routes.yml` says what each endpoint *should* require, decided from
`sync/configs/config.yml` (the role→permission vocabulary `sync` pushes to
Logto), from the documented product behaviour, and from what the endpoint does.
It is never derived from the middleware chain in `main.go`. If the two disagree,
that disagreement is the finding — which is exactly what happened on the first
run. Copying the wiring into the spec would turn the suite into a tautology that
passes no matter how wrong the code is.

Where the real model departs from the declared one, the departure is written down
in `model.yml` with the code that causes it and what should change. Reviewed
exception, not silent adjustment.

## The four layers

| Layer | Question | Checks |
|---|---|---|
| `gate` | May this persona CALL this endpoint? | 170 routes × 16 matrix personas + anonymous |
| `scope` | May it reach THIS OBJECT through it? | hand-written cross-organization scenarios |
| `apps` | Which third-party apps is it offered? | 4 apps × 23 personas, portal vs `access_control` |
| `special` | Rules enforced by middleware or credential type | self-modification, API key masks, impersonation |

### gate — non-destructive by construction

Path parameters are filled with an id that matches nothing and write verbs get a
body that cannot validate, so an authorized call lands on 404/400 *after* the
gate ran. Nothing real is ever touched, which is what makes the matrix safe to
re-run on every change. How a response is read:

| Response | Meaning |
|---|---|
| `403 "insufficient permissions"` | the RBAC middleware refused — a route-level gate |
| `403` anything else | a handler refused — deeper, object-level |
| `404` / `400` | the gate passed; the object or the body did not |
| `2xx` | the gate passed |
| `404 "api not found"` | no endpoint matched — never a pass, the spec path is wrong |

A persona that should be blocked but gets 404 means the route has no gate for it:
the request reached the handler. Whether data actually leaks is then the scope
layer's question.

Verdicts: **FAIL-OPEN** (should have been blocked, was not — the class that leaks
data), **FAIL-CLOSED** (entitled but blocked — a broken promise, not a leak),
**REVIEW** (a human call: a 5xx hiding the answer, or a refusal from an
unexpected place). REVIEW alone does not fail the run; either FAIL does.

### scope — where isolation is actually proved

The fixture has siblings and two branches, because isolation can only be shown
against an organization at the same level on the other side of the tree:

```
owner ── d1 ── d1r1 ── d1r1c1, d1r1c2
         │     └ d1r2 ── d1r2c1
         └ d2 ── d2r1 ── d2r1c1
```

`created_by` decides the hierarchy: organization scope walks
`custom_data.createdBy`, so every org is created *by a user of its intended
parent*. Creating everything as the owner would produce a flat tree and the whole
layer would prove nothing.

`expect: denied` accepts 403 or 404 — hiding existence is a legitimate refusal.
`must_not_contain` catches leakage through lists, aggregates, filters and exports
and runs even on a 200. Matching is by token boundary, not substring: the fixture
names nest on purpose (`authz-d1r1` is a prefix of `authz-d1r1c1`), and plain
substring matching reports a customer's own name as its parent leaking.

## Adding an endpoint

`./apitool authz coverage` fails when `main.go` registers a route that
`routes.yml` does not mention, so a new endpoint cannot ship without a declared
intent. Add an entry, decide `intent` from the permission vocabulary rather than
from the middleware you just wrote, and give the probe whatever it needs
(`ids`, `body`, `query`) to land on the authorization decision instead of on a
validation error. If the endpoint is object-scoped by design, say so with
`object_scoped: true` and add a scope scenario — that is where its real guarantee
lives.

Available placeholders in `scenarios.yml`, `special.yml` and route
`query`/`body`/`ids`: `{org:KEY}`, `{org_name:KEY}`, `{user:KEY}`,
`{user_email:KEY}`, `{system:KEY}`, `{system_name:KEY}`, `{my_org}`,
`{my_user}`, `{bogus}`.

## What the suite cannot reach

- **Owner-organization personas other than the bootstrap owner.** `POST /api/users`
  refuses them ("managed by the system") and `sync init` seeds exactly one. So
  owner/Admin, owner/Support, owner/Backoffice and owner/Reader are untested.
  Note what that implies: since `effective = org_permissions ∪ user_permissions`,
  any user of the Owner organization would hold
  `destroy:distributors|resellers|customers` whatever its technical role.
- **Applications.** They are derived from inventory pushed through `collect`, not
  created through the API, so application scope is covered only with
  nonexistent ids. Pushing an inventory in `provision` would close this.
- **Backup download and alert silences on real objects.** Both need real data
  (an S3 backup, a firing alert); the endpoints are covered, the object-level
  cases are not.
- **The positive impersonation path.** Deliberately: it would leave live sessions
  behind. Only the refusals are asserted.
- **`collect`'s own endpoints.** A separate service with a separate credential
  model (system key/secret over Basic Auth).

## Findings from the first full run

3186 checks, all expectations holding. Details in `authz-report.json`. Findings 2
and 3 below are **fixed**; finding 1 turned out not to be a defect.

### 1. Cross-hierarchy impersonation — INTENDED, spec corrected

The first run flagged this as the worst hole in the platform: `POST /api/impersonate`
checks the `impersonate:users` permission and the target's consent but **not**
whether the target is inside the caller's hierarchy, so a Super Admin of a
reseller reached the sibling reseller, its own distributor, and a customer in
another branch — all 200.

That is the design, not a defect. Nethesis Italia is itself a *distributor*, and
its staff must be able to support users of other distributors and resellers. The
feature therefore has two barriers, and the hierarchy is deliberately not one of
them:

1. the target must have consented;
2. the caller must hold `impersonate:users`, which only the **Super Admin** role
   carries — and only the Owner organization may assign that role.

**Barrier 2 is what protects the whole feature**, so that is where the suite now
asserts. Verified: a distributor Admin, a reseller Admin, a customer Admin and a
Backoffice user are all refused with `insufficient privileges to assign this role`
when promoting a user to Super Admin *or* creating one — including when the role
id is supplied directly rather than picked from `GET /roles`, which only hides it.
Those seven scenarios live in `scenarios.yml`; the three positive impersonation
cases in `special.yml` assert the reach itself, so adding a hierarchy check would
surface as a regression in legitimate support access instead of a silent change.

This is a good illustration of why expectations here are hand-written: the suite
was right that the code allows platform-wide impersonation, and wrong about
whether it should. What it could not know, it forced someone to state.

**Open question worth a decision.** Reach does not depend on where the Super Admin
sits: the third case is a Super Admin of a *customer* organization reaching
another branch entirely. The stated intent is "members of Nethesis Italia, which
is a distributor", so containment today is process (few people, owner-granted),
not code. If that should be enforced, the narrowest change is to require the
impersonator's organization to be the Owner or a distributor — leaving consent and
the role gate exactly as they are. Until decided, the suite documents the current
reach as intended.

Two real defects did surface alongside it, both still open. Neither is an
authorization hole; both are ways the feature can strand an operator or hide what
is happening.

**a. `DELETE /impersonate` contradicts `/status` on the same token.** Three
endpoints answer "is there a session?" from two different sources:

| Endpoint | Reads from | With the impersonator's own token |
|---|---|---|
| `GET /impersonate/status` | Redis, by impersonator id | sees it: *"currently impersonating"* — **and returns a fresh impersonation token** |
| `POST /impersonate` (409 guard) | Redis, by impersonator id | sees it: *"you already have an active session"* |
| `DELETE /impersonate` (exit) | the **token's** own claims (`is_impersonated`) | does not see it: *400 "not currently impersonating a user"* |

Ending a session therefore takes two calls: `GET /status` to recover an
impersonation token, then `DELETE` with that token. That path is deliberate —
`/status` documents it for the page-reload case — so nobody is ever permanently
stuck. What is wrong is only that the same credential gets contradictory answers
from two endpoints, and nothing at the exit endpoint says the dance is required. A
caller that meets the 400 reasonably concludes there is no session.

The suite closes its own sessions with the token `POST /impersonate` returns, so it
never needs the dance.

Not fixed: the one-call exit was implemented and then reverted on request, since
it touches the impersonation flow that is being reconsidered as a whole.

**b. `GET /impersonate/sessions` answers a different question than its name.**
`GetUserSessions` filters `WHERE impersonated_user_id = $1`, so it returns the
sessions in which the caller **was impersonated** — the target's view. Verified:
the target sees 8 sessions, the Super Admin who opened them sees 0, and the audit
table itself is complete and correct. Nothing is lost; the endpoint is simply
named and documented as *"all impersonation sessions for current user"* while
meaning *"sessions where I was the target"*. The consequence is that there is no
way to ask the two questions an operator or an auditor actually has: which
sessions did I open, and who is impersonating whom right now.

### 2. `GET /api/users/:id` did not walk the hierarchy — FAIL-CLOSED, FIXED

A reseller Admin saw the users of its own customers in `GET /api/users` — 11 of
them — but `GET /api/users/:id` on one of those very users answered
`403 "access denied to user"`. `methods/users.go` compared
`targetOrgID == user.OrganizationID` only, while the list uses the
hierarchy-aware `helpers.AppendOrgFilter`: two authorization implementations for
one resource, and the detail one was wrong. The code said as much — *"Additional
logic needed to check customer organizations"*.

Fixed by adding `local.CanReadUser`, the missing member of the
`CanCreateUser`/`CanUpdateUser`/`CanDeleteUser`/`CanSuspendUser` family, and
calling it from `GetUser`. The invariant that made this a bug is now pinned by
`TestLocalUserService_CanReadUserMatchesCanUpdateUser`: whoever may update a user
must be able to read it.

A second defect surfaced in the same block: the "users can always see themselves"
branch compared the `:id` parameter (a **logto_id**) against `user.ID` (the
**local database id**), so it never matched — self-read worked only incidentally,
because the caller is in its own organization. It now compares `LogtoID`, as
`PreventSelfModification` does.

### 3. Rebranding delete handlers mapped every error to one status — FIXED

`DELETE /api/rebranding/:org_id/products/:product_id` answered 500 when there was
simply nothing to delete, which is what the suite caught. The sibling
`DELETE .../:asset` had the mirror defect and the suite could *not* catch it: it
collapsed every error into 404, so a storage failure was reported as "asset not
found" — indistinguishable from a legitimate miss.

Both came from the service returning an untyped `fmt.Errorf` for "no rows
affected". Fixed with sentinel errors (`ErrRebrandingAssetsNotFound`,
`ErrInvalidRebrandingAsset`, following the existing `ErrAPIKey*`/`ErrPromote*`
convention) so the handlers answer 404 for a miss, 400 for an invalid asset name,
and 500 only for real failures — which are now the only case that logs.

### Two design observations, not defects

- **The `Reader` exception lives in Go, not in config.yml.** Organization
  permissions come from the organization's role, so on the declared model every
  Reader of a Distributor organization would hold `manage:resellers`.
  `filterManagePermissionsForReader` subtracts exactly three permission strings
  for exactly one role name. The behaviour is right; its location means a new
  organization-level permission is granted to Readers by default, and a read-only
  role not literally named "Reader" gets full manage rights. See `model.yml`.
- **Logto does not enforce `access_control`.** Portal filtering matched
  `config.yml` exactly for all 23 personas × 4 apps, but that filter is
  visibility, not authorization: a persona the portal hides can still complete
  SSO straight through the app's login URL. Recorded per-app as
  `idp_enforced: false`; flip it the day Logto is made to enforce it and the
  suite will start requiring refusals.

### Known latent issue, not addressed

`isOrganizationInHierarchy` guards `database.DB == nil` on the `owner` branch but
not on the distributor/reseller ones, so with no database those paths nil-deref
rather than denying. Production always has a database, so it is latent; it showed
up because a unit test reached those branches. Left alone deliberately — it is a
robustness fix in a hot authorization path and deserves its own change.

### What held

Everything else. 170 endpoints × 16 personas with no unintended access; the
authentication boundary on all 165 non-public routes; cross-organization
isolation across reads, writes, lists, aggregates, filters, exports, password
resets, system inventory, alert silences, rebranding and VAT existence
disclosure; the `?organization_id=` override, which widens results for the owner
and for nobody else; API keys, capped by both their read/write mode and their
owner's live permissions and refused outright on session-bound routes; and the
Super Admin role gate, which is the single thing standing between a distributor
admin and platform-wide impersonation.
