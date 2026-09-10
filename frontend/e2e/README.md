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
npm run test:e2e            # the local suite: setup + fullstack
npm run test:e2e:smoke      # read-only, against QA
npm run test:e2e:ui         # interactive
npx playwright show-report  # last run: screenshots, video, traces
```

Two configurations, one per target. `playwright.config.ts` holds the local suite and is what a bare
`playwright test` picks up; `playwright.config.smoke.ts` holds the deployed one, and
`test:e2e:smoke` selects it with `-c`. Options common to both live in `e2e/playwright.shared.ts`.
Neither needs a variable set to do the right thing.

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
as a sign-in redirect URI, so the OIDC callback fails on any other port. It is hardcoded in
`playwright.config.ts` for that reason. Playwright starts the server itself when nothing is
listening there, and reuses whatever is.

If you already have one running, note that a plain `npm run dev` does **not** set `VITE_E2E`, which
is what suppresses query auto-refetch and the Pinia Colada devtools panel — both of which race
assertions. Prefer:

```bash
npm run dev:e2e
```

### Running against a deployed environment

```bash
npm run test:e2e:smoke
```

That targets `https://qa.my.nethesis.it`, the configuration's default. Point it elsewhere — a
review environment, the compose stack behind the proxy — with `E2E_SMOKE_BASE_URL`, the only
base-URL variable the suite reads.

Only this configuration may point at a deployed environment, and nothing the local suite reads can
be made to: `playwright.config.ts` hardcodes `http://localhost:5173`. That matters because smoke is
read-only by construction — QA shares a database and a Logto tenant with real users — while the
`fullstack` specs create and delete organizations.

Without `E2E_SMOKE_EMAIL` / `E2E_SMOKE_PASSWORD` the authenticated checks skip and only the public
surface is covered. Those credentials belong to one dedicated account in the target environment's
tenant, not to any persona in the apitool registry — the registry describes the local fixture.

Rather than retyping them, put them in `frontend/.env.e2e`, which `e2e/playwright.shared.ts` loads
when it exists and `.gitignore` already covers:

```bash
E2E_SMOKE_EMAIL=e2e@example.com
E2E_SMOKE_PASSWORD=...
# E2E_SMOKE_BASE_URL=https://qa.my.nethesis.it   # only to override the default
# E2E_USER_MAIL=you@example.com                  # base address for users the fullstack specs create
```

The smoke suite needs an origin that serves both the application and `/backend/api`, which means a
deployed environment or the compose stack behind the proxy. A bare Vite dev server serves neither,
and the health check says so.

Note QA is suspended outside Mon–Fri 08:00–22:00 Europe/Rome by `qa-night-schedule.yml`.

## In CI

| Workflow        | Trigger                                               | What it runs                                      |
| --------------- | ----------------------------------------------------- | ------------------------------------------------- |
| `e2e-main.yml`  | every push to a PR and to `main`, manual, weekly cron | The `fullstack` project against the compose stack |
| `e2e-smoke.yml` | push to `main`, manual                                | `playwright.config.smoke.ts` against QA           |

Docs-only pushes are skipped on both `e2e-main.yml` triggers. Its concurrency group is global and
never cancels, so a burst of pushes leaves the commits in between untested rather than queueing.

`e2e-main.yml` brings the stack up with `docker-compose.e2e.yml` layered on top, which publishes
the proxy on 5173 (the registered redirect origin) and builds the frontend with `VITE_E2E`. It
passes no base URL: `webServer.reuseExistingServer` finds the proxy already listening there and
starts no dev server. It
writes its own `.api-registry.json` from secrets — `apitool init` is interactive — and then runs
`authz provision`, tearing the fixture down in an `always()` step so a failed run leaves nothing in
the tenant.

Its `E2E_*` secrets point at a tenant dedicated to CI. The fixture is not per-run — `prefix` in
`backend/authz/fixture.yml` fixes the organization keys and persona addresses — so pointing them at
the tenant people develop against would have CI and a local `authz provision` fighting over the same
Logto users. See `.github/workflows/README.md`.

`e2e-smoke.yml` polls `/api/health` until it reports the merge commit, since QA is deployed by
Render rather than by Actions. On timeout it warns and skips instead of failing, so a suspended QA
does not read as a regression.

Both upload `playwright-report/` and `test-results/` on failure: screenshots, video and traces.

## Layout

| Path                   | Purpose                                                                                          |
| ---------------------- | ------------------------------------------------------------------------------------------------ |
| `fixtures/personas.ts` | Reads the apitool registry; exposes `owner`, `matrixPersonas`, `persona(key)`, `fixtureOrg(key)` |
| `fixtures/i18n.ts`     | Resolves interface copy by translation key, so a reworded label updates the selector             |
| `fixtures/rows.ts`     | Matches a table row by whole name — the fixture's names nest, so substrings lie                  |
| `setup/auth.setup.ts`  | Signs personas in through the real Logto form, saves `storageState`                              |
| `fullstack/`           | Need a local backend and a provisioned fixture                                                   |
| `smoke/`               | Read-only specs against a deployed environment                                                   |
| `.auth/`               | Saved sessions, one file per persona. Gitignored — they are live Logto sessions                  |

What each `fullstack/` spec answers:

| Spec                    | Question                                                                                 |
| ----------------------- | ---------------------------------------------------------------------------------------- |
| `login.spec.ts`         | Does a saved session boot straight in, and an empty one get bounced to Logto?            |
| `rbac.spec.ts`          | Is each persona offered the right navigation, and refused every section it may not read? |
| `controls.spec.ts`      | Inside a page it may open, is it offered the right actions?                              |
| `scoping.spec.ts`       | Is the table it reads scoped to its own branch of the hierarchy?                         |
| `session.spec.ts`       | Does the session survive a reload, and does a refusal leave the rest of it usable?       |
| `organizations.spec.ts` | Can a company be created and edited through the interface?                               |
| `systems.spec.ts`       | Do the list, the detail view and the registration handshake behave?                      |
| `users.spec.ts`         | Can a user be created, re-roled and archived?                                            |

Only `organizations.spec.ts`, `systems.spec.ts` and `users.spec.ts` write anything; the rest are
read-only.

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
  Logto refresh token, and rotating it in parallel looks like token theft. It stays quiet while the
  access token the setup project minted is still valid, so the risk grows with how long a run takes
  rather than with how many specs there are — and more than one spec now iterates `matrixPersonas`,
  so two workers on two files really do drive the same persona at once. Partition personas across
  projects before raising the count.
- **No `data-testid` yet.** Prefer `getByRole` / `getByLabel`, resolving the text through
  `fixtures/i18n.ts` so a reworded label updates the selector for free. The app is translated
  (en/it) and that helper reads the English catalogue only, so the locale is pinned in two places
  and both matter: `locale: 'en-US'` in `playwright.config.ts`, and `auth.setup.ts` clearing the
  `preferences-<email>` localStorage entry, whose saved `locale` would otherwise outrank the
  browser's and travel inside `storageState`.
- **Never `waitForTimeout`.** The token refresh timer and the `visibilitychange` handler put traffic
  on the wire mid-test; web-first assertions (`expect(locator).toBeVisible()`) retry, sleeps do not.
- **Give a negative assertion a positive control.** `expect(x).toHaveCount(0)` after an interaction
  passes just as well when the interaction did not happen, so assert something that must be there
  first — an element the state under test does not affect. Same reason the RBAC spec refuses to pass
  for a persona whose token carried no permissions at all.
- **Do not wait for something already on screen.** Filtering a list and then waiting for the target
  row proves nothing: it was visible before the first keystroke. Wait for the narrowed state.

## Cleanup

```bash
cd backend && ./apitool authz teardown --yes
```
