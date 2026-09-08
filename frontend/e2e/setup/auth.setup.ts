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

import { test as setup, expect } from '@playwright/test'
import { mkdirSync } from 'node:fs'
import { dirname } from 'node:path'
import { matrixPersonas, owner, storageStatePath, type Persona } from '../fixtures/personas'

/**
 * The owner plus the RBAC matrix — one persona per (organization role x
 * technical role) pair. Each costs one real Logto sign-in, run serially so a
 * burst of parallel logins never looks like abuse to the tenant.
 */
const personas: Persona[] = [owner, ...matrixPersonas]

async function signIn(page: import('@playwright/test').Page, who: Persona) {
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
 * The give-away is a burst of Vue "Missing ref owner context" warnings, which a
 * freshly started server never emits. Catch it here rather than in twenty
 * confusing spec failures.
 */
setup('the dev server is serving current code', async ({ page }) => {
  const hoistedRefWarnings: string[] = []

  page.on('console', (message) => {
    if (message.text().includes('Missing ref owner context')) {
      hoistedRefWarnings.push(message.text())
    }
  })

  await page.goto('/login')
  await page.waitForTimeout(2_000)

  expect(
    hoistedRefWarnings,
    'The dev server on this port is serving a stale module graph: component-library ' +
      'refs are undefined, so dropdowns and drawers will not open and the specs ' +
      'would report product bugs that do not exist. Restart it — `npm run dev:e2e`.',
  ).toEqual([])
})

for (const who of personas) {
  setup(`authenticate ${who.key}`, async ({ page }) => {
    const file = storageStatePath(who.key)
    mkdirSync(dirname(file), { recursive: true })

    await signIn(page, who)
    await page.context().storageState({ path: file })
  })
}
