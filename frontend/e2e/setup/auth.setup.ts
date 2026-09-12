//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Signs personas in through the real Logto sign-in form and saves the browser
 * state, so specs start authenticated instead of paying for a redirect dance
 * each time.
 *
 * Only `storageState` is saved, which covers cookies and localStorage — the
 * Logto SDK session lives there. The application's own JWT pair is deliberately
 * *not* carried over: it sits in sessionStorage (see `stores/login.ts`), which
 * storageState does not capture, and replaying one saved refresh token from
 * several workers would trip the backend's rotation/reuse detection. Each spec
 * therefore re-runs `POST /api/auth/exchange` on boot for a fresh pair, which
 * is both race-free and the real code path.
 */

import { test as setup, expect, type Page } from '@playwright/test'
import { mkdirSync } from 'node:fs'
import { dirname } from 'node:path'
import { matrixPersonas, owner, storageStatePath, type Persona } from '../fixtures/personas'
import { t } from '../fixtures/i18n'

/**
 * The owner plus the RBAC matrix — one persona per (organization role x
 * technical role) pair. Each costs one real Logto sign-in, run serially so a
 * burst of parallel logins never looks like abuse to the tenant.
 */
const personas: Persona[] = [owner, ...matrixPersonas]

/**
 * Drop the per-user preference blob before the session is saved.
 *
 * `savePreference` from the component library keeps user preferences in a
 * localStorage entry named `preferences-<email>`, and `getBrowserLocale`
 * (`src/i18n/index.ts`) prefers the `locale` it finds there over anything the
 * browser negotiates. localStorage is exactly what `storageState` captures and
 * `e2e/.auth/*.json` keeps between runs, so one persona that ever had Italian
 * selected would silently break every selector in the suite — each of which is
 * resolved through the English catalogue by `fixtures/i18n.ts`. Clearing it
 * also makes the collapsed/expanded menu state deterministic.
 */
async function clearStoredPreferences(page: Page) {
  await page.evaluate(() => {
    for (const key of Object.keys(localStorage)) {
      if (key.startsWith('preferences-')) {
        localStorage.removeItem(key)
      }
    }
  })
}

async function signIn(page: Page, who: Persona) {
  // "/" redirects to /dashboard, the router guard bounces an unauthenticated
  // visitor to /login, and LoginView immediately calls signIn() — which is a
  // full-page navigation to the Logto-hosted form on another origin.
  await page.goto('/')
  await page.waitForURL(/\/sign-in/, { timeout: 60_000 })

  await page.locator('input[name="identifier"]').fill(who.email)
  await page.locator('input[name="password"]').fill(who.password)
  await page.locator('button[type="submit"]').click()

  // Back on our origin: /login-redirect completes the OIDC callback and pushes
  // to the saved deep link, or the dashboard.
  await page.waitForURL((url) => url.pathname === '/dashboard', { timeout: 60_000 })

  // The dashboard renders before user info arrives. Wait for the exchange to
  // land, otherwise the saved state can be a half-built session.
  await expect(page.locator('#app')).toBeVisible()
  await page.waitForFunction(() => sessionStorage.getItem('my_jwt') !== null, null, {
    timeout: 30_000,
  })

  await clearStoredPreferences(page)
}

for (const who of personas) {
  setup(`authenticate ${who.key}`, async ({ page }) => {
    const file = storageStatePath(who.key)
    mkdirSync(dirname(file), { recursive: true })

    await signIn(page, who)
    await page.context().storageState({ path: file })
  })
}

/**
 * Fails fast when the dev server being used is serving a stale module graph.
 *
 * `webServer.reuseExistingServer` lets the suite attach to a dev server that is
 * already running, which is what makes local iteration quick. The catch: a
 * long-running Vite server that hot-reloaded through an edit to the app entry
 * (`main.ts`) can end up in a broken state where component-library refs come
 * back undefined — dropdowns and side drawers stop opening, while navigation
 * and the rest of the page look perfectly healthy. Every symptom then reads as
 * a product bug, and a suite that trusts it reports nonsense.
 *
 * So open one. A drawer that appears is the only evidence that means anything
 * here, and it costs one page load: watching the console for the Vue "Missing
 * ref owner context" warnings the broken state emits proves nothing on a page
 * that mounts no such component, and needs a sleep to do even that.
 *
 * Declared after the sign-in loop so the owner's session already exists.
 */
setup.describe('the dev server is serving current code', () => {
  setup.use({ storageState: storageStatePath(owner.key) })

  setup('a side drawer opens', async ({ page }) => {
    await page.goto('/distributors')
    await page.getByRole('button', { name: t('distributors.create_distributor') }).click()

    const drawer = page.locator('form').filter({ has: page.getByLabel(t('organizations.name')) })

    await expect(
      drawer,
      'The create drawer did not open. If this is a dev server that has been running for a ' +
        'while, it is probably serving a stale module graph: component-library refs come back ' +
        'undefined, so dropdowns and drawers stay shut and the specs would report product bugs ' +
        'that do not exist. Restart it — `npm run dev:e2e`.',
    ).toBeVisible({ timeout: 30_000 })
  })
})
