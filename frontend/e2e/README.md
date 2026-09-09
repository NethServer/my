# End-to-end suite

Playwright specs that drive the real application in a browser against a real backend. They
complement, rather than duplicate, the API-level coverage in `backend/authz/`: that suite proves the
API refuses the wrong caller, these prove the UI hides the button, scopes the table and renders the
right navigation for that same persona.

## Prerequisites

1. **A backend on `:8080`** with Postgres and Redis behind it:

   ```bash
   cd backend && make dev-up && make db-migrate && make run
   ```

2. **A provisioned fixture.** The suite reads its personas — emails _and_ passwords — from the
   registry `apitool` writes, so it needs no fixture of its own:

   ```bash
   cd backend && make apitool && ./apitool authz provision   # idempotent
   ```

   This creates the organization hierarchy and one user per (organization role × technical role)
   pair, fixing a password on each via the Logto Management API. Credentials land in
   `backend/.api-registry.json` (gitignored, mode 0600).

## Running

```bash
cd frontend
npm run test:e2e                      # everything
npm run test:e2e -- --project=fullstack
npm run test:e2e:ui                   # interactive
npx playwright show-report            # last run: screenshots, video, traces
```

A single spec, watching it happen:

```bash
npx playwright test e2e/fullstack/login.spec.ts --project=fullstack --no-deps --headed --debug
```

`--no-deps` is what makes that one spec the only thing that runs. A file
argument does not filter a project's dependencies, so without it the `setup`
project runs whole first — and since `--debug` pauses before the first action of
the first test in the queue, the browser opens on a blank page belonging to a
persona login rather than to the spec you named. It reuses the sessions already
in `e2e/.auth/`, so run the suite (or the `setup` project) at least once first.

### The dev server must be on port 5173

The Logto application the fixture was provisioned against only accepts `http://localhost:5173/...`
as a sign-in redirect URI, so the OIDC callback fails on any other port. Playwright starts the
server itself when nothing is listening there.

If you already have one running, note that a plain `npm run dev` does **not** set `VITE_E2E`, which
is what suppresses query auto-refetch and the Pinia Colada devtools panel — both of which race
assertions. Prefer:

```bash
npm run dev:e2e
```

### Running against a deployed environment

```bash
E2E_BASE_URL=https://qa.my.nethesis.it \
E2E_SMOKE_EMAIL=... E2E_SMOKE_PASSWORD=... \
npm run test:e2e -- --project=smoke
```

Only the `smoke` project may point at a deployed environment. It is read-only by construction: QA
shares a database and a Logto tenant with real users, and the `fullstack` specs create and delete
organizations.

Without `E2E_SMOKE_EMAIL` / `E2E_SMOKE_PASSWORD` the authenticated checks skip and only the public
surface is covered. Those credentials belong to one dedicated account in the target environment's
tenant, not to any persona in the apitool registry — the registry describes the local fixture.

The smoke project needs an origin that serves both the application and `/api`, which means a
deployed environment or the compose stack behind the proxy. A bare Vite dev server serves neither,
and the health check says so.

Note QA is suspended outside Mon–Fri 08:00–22:00 Europe/Rome by `qa-night-schedule.yml`.

## In CI

| Workflow        | Trigger                             | What it runs                                      |
| --------------- | ----------------------------------- | ------------------------------------------------- |
| `e2e-main.yml`  | push to `main`, manual, weekly cron | The `fullstack` project against the compose stack |
| `e2e-smoke.yml` | push to `main`, manual              | The `smoke` project against QA                    |

`e2e-main.yml` brings the stack up with `docker-compose.e2e.yml` layered on top, which publishes
the proxy on 5173 (the registered redirect origin) and builds the frontend with `VITE_E2E`. It
writes its own `.api-registry.json` from secrets — `apitool init` is interactive — and then runs
`authz provision`, tearing the fixture down in an `always()` step so a failed run leaves nothing in
the tenant.

`e2e-smoke.yml` polls `/api/health` until it reports the merge commit, since QA is deployed by
Render rather than by Actions. On timeout it warns and skips instead of failing, so a suspended QA
does not read as a regression.

Both upload `playwright-report/` and `test-results/` on failure: screenshots, video and traces.

## Layout

| Path                   | Purpose                                                                         |
| ---------------------- | ------------------------------------------------------------------------------- |
| `fixtures/personas.ts` | Reads the apitool registry; exposes `owner`, `matrixPersonas`, `persona(key)`   |
| `setup/auth.setup.ts`  | Signs personas in through the real Logto form, saves `storageState`             |
| `fullstack/`           | Mutating specs. Need a local backend and a provisioned fixture                  |
| `smoke/`               | Read-only specs against a deployed environment                                  |
| `.auth/`               | Saved sessions, one file per persona. Gitignored — they are live Logto sessions |

## How authentication works

Playwright drives the actual Logto sign-in form with the persona's registry password, so the suite
exercises the real OIDC path and needs no test-only code in the application.

Only `storageState` is saved, which covers cookies and localStorage — where the Logto SDK keeps its
session. The application's own JWT pair is deliberately **not** carried between tests: it lives in
`sessionStorage` (`src/stores/login.ts`), which `storageState` does not capture anyway, and the
backend rotates the refresh token on every use and treats a reused one as theft. Each spec therefore
re-runs `POST /api/auth/exchange` on boot for a fresh pair — race-free, and the real code path.

## Gotchas

- **Keep `workers` low.** Several workers driving one persona's saved session share that persona's
  Logto refresh token, and rotating it in parallel looks like token theft. Partition personas across
  projects before raising the count.
- **No `data-testid` yet.** Prefer `getByRole` / `getByLabel`. The app is translated (en/it), so
  selectors that match visible text break under a different locale — pin the locale rather than
  matching prose.
- **Never `waitForTimeout`.** The token refresh timer and the `visibilitychange` handler put traffic
  on the wire mid-test; web-first assertions (`expect(locator).toBeVisible()`) retry, sleeps do not.

## Cleanup

```bash
cd backend && ./apitool authz teardown --yes
```
