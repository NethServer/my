# GitHub Actions

## PR Preview Environment — comment commands

Comment one of these on a PR to control its Render preview environment (the QA
services with `pullRequestPreviewsEnabled: true` in `render.yaml`). Render names
each preview instance `my-<component>-qa PR #<PR_NUMBER>` (the
`my-<component>-qa-pr-<N>` form is only the hostname). Services not created for a
given PR are skipped.

| Comment | Workflow | Effect |
|---------|----------|--------|
| `update deploy` | `pr-build-trigger.yml` | Touches `.render-build-trigger` files and pushes, forcing a fresh rebuild of the latest commit. |
| `down deploy` | `pr-preview-suspend.yml` | Suspends the preview services (`POST /v1/services/{id}/suspend`) to stop paying for compute while the PR stays open. |
| `up deploy` | `pr-preview-resume.yml` | Resumes the preview services (`POST /v1/services/{id}/resume`), bringing back the last build. |

All three are restricted to `OWNER` / `MEMBER` / `COLLABORATOR` comment authors.

**Important:** a suspended service does **not** wake up on a push — Render ignores
autodeploy while suspended. To bring a suspended env back:

1. `up deploy` → resume (restores the last build).
2. `update deploy` → rebuild the latest commit (optional, only if you pushed while suspended).

`down deploy` / `up deploy` require the `RENDER_API_KEY` secret (see below).

## Logto Redirect URI Management

These GitHub Actions automatically manage redirect URIs in your Logto application configuration for Pull Request deployments on Render.

## Workflows

### 1. `pr-redirect-uris-add.yml`
**Trigger**: When a PR is opened or reopened
**Purpose**: Adds redirect URIs for the PR's Render deployments to Logto

### 2. `pr-redirect-uris-remove.yml`
**Trigger**: When a PR is closed or merged
**Purpose**: Removes redirect URIs for the PR's Render deployments from Logto

## End-to-end suite

### `e2e-main.yml`
**Trigger**: Every push to a pull request and to `main` (docs-only pushes skipped), manual dispatch,
weekly cron
**Purpose**: Runs the browser suite (`frontend/e2e/`, `--project=fullstack`) against the full
compose stack, with personas provisioned by `apitool authz provision`

Deliberately separate from `ci-main.yml`: that workflow answers in seconds and gates every branch,
while this builds four images, boots six services and mutates a shared Logto tenant, so it takes
minutes. It runs per push to a pull request so a regression is attributed to the commit that caused
it rather than to a batch of merges.

Its concurrency group is **global and queues rather than cancels** — a run cancelled after
provisioning would abandon real organizations and users in the tenant. GitHub keeps at most one run
pending per group, so under a burst of pushes the commits in between are simply not tested; that is
the accepted cost of never cancelling. The weekly cron is a drift canary for breakage with no commit
behind it, such as a tenant setting changed by hand.

### `e2e-smoke.yml`
**Trigger**: Push to `main`, manual dispatch
**Purpose**: Read-only checks against QA (`--project=smoke`), covering the deployment-configuration
failures the full-stack job cannot see

QA is deployed by Render rather than by Actions, so the job asks the Render API (via the existing
`RENDER_API_KEY`) for a deploy of the merge commit, waits for it to go `live`, and then confirms the
backend actually answers. The health endpoint cannot identify the build: Render builds QA from
source and nothing passes `COMMIT`, so it reports `"unknown"` permanently. A failed Render deploy
fails the job; an environment that never comes up only warns and skips, since `qa-night-schedule.yml`
suspends QA outside Mon–Fri 08:00–22:00 Europe/Rome.

See `frontend/e2e/README.md` for the suite itself.

## Required GitHub Secrets

Add these secrets to your repository settings (`Settings > Secrets and variables > Actions`):

| Secret Name | Description | Example Value |
|-------------|-------------|---------------|
| `RENDER_API_KEY` | Render API key (Account Settings → API Keys). Used by `down deploy` / `up deploy`. | `rnd_xxxxxxxxxxxx` |
| `LOGTO_BASE_URL` | Your Logto instance base URL | `https://your-tenant-id.logto.app` |
| `LOGTO_M2M_CLIENT_ID` | Machine-to-Machine application client ID | `abcd1234efgh5678ijkl` |
| `LOGTO_M2M_CLIENT_SECRET` | Machine-to-Machine application secret | `your-secret-here` |
| `LOGTO_FRONTEND_APP_ID` | Frontend application ID to update | `frontend-app-id-here` |

### End-to-end suite

These must point at a Logto tenant **dedicated to CI** — neither QA nor production, and not the
tenant people develop against.

Two reasons. The full-stack specs create and delete organizations, and QA and production share a
database and a tenant with real users. And the fixture is not per-run: `prefix` in
`backend/authz/fixture.yml` fixes the organization keys and persona addresses, so a CI run and
somebody's local `apitool authz provision` on the same tenant fight over the same Logto users. (What
each side deletes is safely scoped — the specs refuse any name outside the `e2e-` prefix and
`authz teardown` only removes what its own registry records — so the failure mode is a collision
during provisioning, not lost data.)

| Secret Name | Description | Example Value |
|-------------|-------------|---------------|
| `E2E_LOGTO_ENDPOINT` | Logto endpoint for the e2e tenant | `https://your-tenant.logto.app` |
| `E2E_LOGTO_APP_ID` | SPA application id the fixture is provisioned against | `p18mtn23wn87nvz1tscf7` |
| `E2E_LOGTO_TENANT_ID` | Tenant id, for the backend | `your-tenant-id` |
| `E2E_LOGTO_TENANT_DOMAIN` | Tenant domain, for the backend | `your-tenant.logto.app` |
| `E2E_LOGTO_BACKEND_APP_ID` | M2M application id with Management API access | `abcd1234efgh5678ijkl` |
| `E2E_LOGTO_BACKEND_APP_SECRET` | M2M application secret | `your-secret-here` |
| `E2E_JWT_SECRET` | Signing key for the stack under test (min 32 chars) | `a-32-char-or-longer-random-string` |
| `E2E_OWNER_EMAIL` | Owner account `apitool` acts as | `owner@example.com` |
| `E2E_OWNER_PASSWORD` | Owner account password | `your-password-here` |
| `E2E_SMOKE_EMAIL` | Dedicated read-only account in the **QA** tenant | `e2e@example.com` |
| `E2E_SMOKE_PASSWORD` | That account's password | `your-password-here` |

Without the two `E2E_SMOKE_*` values the smoke job still runs, covering only the public surface.

There are deliberately no `SMTP_*` secrets here. Creating a user makes the backend send a welcome
email with a temporary password, and the full-stack suite creates one per run; the job writes no
mail configuration, so nothing is sent. Do not add any.

## Setup Instructions

1. **Create M2M Application in Logto**:
   - Go to Logto Admin Console → Applications → Machine-to-Machine
   - Create new app with Management API permissions
   - Copy the Client ID and Secret

2. **Find Frontend Application ID**:
   - Go to Logto Admin Console → Applications
   - Find your frontend application
   - Copy the Application ID from the URL or application details

3. **Add Secrets to GitHub**:
   - Go to your repository → Settings → Secrets and variables → Actions
   - Add all required secrets listed above

4. **Test the Workflow**:
   - Create a test PR to verify the workflow runs
   - Check the PR comments for confirmation
   - Verify redirect URIs are added/removed in Logto Admin Console

## Generated URIs

For each PR, the following redirect URIs are automatically managed:

- **Frontend**: `https://my-frontend-qa-pr-{PR_NUMBER}.onrender.com/login-redirect`
- **Proxy**: `https://my-proxy-qa-pr-{PR_NUMBER}.onrender.com/login-redirect`

Both URIs are added to:
- `redirectUris` (for login redirects)
- `postLogoutRedirectUris` (for logout redirects)

## Error Handling

The workflows include comprehensive error handling:
- Token acquisition failure detection
- API response validation
- Duplicate URI prevention (when adding)
- Missing URI handling (when removing)
- Detailed logging for troubleshooting

## Troubleshooting

**Common Issues**:

1. **401 Unauthorized**: Check M2M app has Management API permissions
2. **404 Not Found**: Verify `LOGTO_FRONTEND_APP_ID` is correct
3. **Token errors**: Verify `LOGTO_BASE_URL`, `LOGTO_M2M_CLIENT_ID`, and `LOGTO_M2M_CLIENT_SECRET`
4. **Invalid resource indicator**: The resource URL is automatically constructed as `{LOGTO_BASE_URL}/api`

**Debug Steps**:
1. Check GitHub Actions logs for detailed error messages
2. Verify all secrets are correctly set
3. Test M2M app permissions in Logto Admin Console
4. Ensure `LOGTO_BASE_URL` is in format `https://your-tenant-id.logto.app` (without trailing slash)