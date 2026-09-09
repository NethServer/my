//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Post-deploy smoke tests against a deployed environment.
 *
 * These catch the one class of failure the full-stack job structurally cannot
 * see: the stack it tests is built by compose from this checkout, with its own
 * configuration, so a wrong environment variable in `render.yaml`, a missing
 * Logto redirect URI, a proxy route that differs from `nginx.conf.local` or a
 * stale image tag are all invisible there and fatal here.
 *
 * Read-only by construction. The target shares a database and a Logto tenant
 * with real users, so nothing here creates, edits or deletes anything — that
 * is what the `fullstack` project is for, against a local stack.
 *
 * Credentials come from the environment rather than the apitool registry: the
 * registry describes the local fixture, while this signs in as one dedicated
 * account provisioned in the deployed environment's tenant.
 *
 *   E2E_BASE_URL=https://qa.my.nethesis.it \
 *   E2E_SMOKE_EMAIL=... E2E_SMOKE_PASSWORD=... \
 *   npm run test:e2e -- --project=smoke
 */

import { test, expect } from '@playwright/test'
import { t } from '../fixtures/i18n'

/**
 * Where the backend sits behind the deployed proxy. NOT `/api`, which the proxy
 * forwards to the separate legacy system (`proxy/nginx.conf:208`) — asserting
 * there would test someone else's service. `render.yaml` sets
 * VITE_API_BASE_URL to this for both QA and production.
 */
const API = '/backend/api'

const email = process.env.E2E_SMOKE_EMAIL
const password = process.env.E2E_SMOKE_PASSWORD

test.describe('public surface', () => {
  test('serves a healthy backend', async ({ request, baseURL }) => {
    const res = await request.get(`${baseURL}${API}/health`)
    expect(res.status()).toBe(200)

    // A deployed environment serves the application and the API from one
    // origin, through the proxy. Pointed at a bare Vite dev server this comes
    // back as index.html, so say so rather than failing on a JSON parse error.
    expect(
      res.headers()['content-type'] ?? '',
      `${baseURL}${API}/health did not return JSON. The smoke project expects an origin ` +
        `that serves both the application and ${API} — a deployed environment, or the ` +
        'compose stack behind the proxy. A Vite dev server serves neither.',
    ).toContain('application/json')

    const body = (await res.json()) as { data: { version: string; commit: string } }
    expect(body.data.version).toBeTruthy()
  })

  test('sends an anonymous visitor to the identity provider', async ({ page }) => {
    await page.goto('/systems')

    // Proves the deployed redirect URI is registered: Logto rejects an
    // unregistered one with an error page instead of the sign-in form.
    await page.waitForURL(/\/sign-in/, { timeout: 60_000 })
    await expect(page.locator('input[name="identifier"]')).toBeVisible()
  })
})

test.describe('authenticated surface', () => {
  test.skip(
    !email || !password,
    'Set E2E_SMOKE_EMAIL and E2E_SMOKE_PASSWORD to the dedicated e2e account for this environment',
  )

  test('signs in, loads the dashboard and signs out', async ({ page }) => {
    await page.goto('/')
    await page.waitForURL(/\/sign-in/, { timeout: 60_000 })

    await page.locator('input[name="identifier"]').fill(email!)
    await page.locator('input[name="password"]').fill(password!)
    await page.locator('button[type="submit"]').click()

    await page.waitForURL((url) => url.pathname === '/dashboard', { timeout: 60_000 })

    // The token exchange completed: this is the application's own JWT, so the
    // backend, the database and the Logto wiring are all reachable.
    await expect
      .poll(() => page.evaluate(() => sessionStorage.getItem('my_jwt')), { timeout: 60_000 })
      .not.toBeNull()

    await expect(page.getByRole('navigation')).toBeVisible()
  })

  test('renders a list page', async ({ page }) => {
    await page.goto('/')
    await page.waitForURL(/\/sign-in/, { timeout: 60_000 })
    await page.locator('input[name="identifier"]').fill(email!)
    await page.locator('input[name="password"]').fill(password!)
    await page.locator('button[type="submit"]').click()
    await page.waitForURL((url) => url.pathname === '/dashboard', { timeout: 60_000 })

    const systems = page.getByRole('navigation').locator('a[href="/systems"]')
    await expect(systems).toBeVisible()

    // Armed before the click: waitForResponse only sees traffic that arrives
    // after it starts listening.
    const listed = page.waitForResponse(
      (r) => r.url().includes(`${API}/systems`) && r.request().method() === 'GET',
      { timeout: 60_000 },
    )
    await systems.click()

    expect((await listed).status()).toBe(200)

    // And then the DOM, because a 200 is not a rendered list: a table that
    // never leaves its skeleton, or a component that throws on the payload,
    // looks identical on the wire. Either a row or the empty state — this
    // environment's inventory is not ours to assume.
    await expect(
      page
        .getByRole('row')
        .first()
        .or(page.getByText(t('systems.no_systems_found'))),
    ).toBeVisible({ timeout: 60_000 })
  })
})
