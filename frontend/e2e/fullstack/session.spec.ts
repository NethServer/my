//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Session behaviour across a reload, and what a refusal does to a live session.
 *
 * `stores/login.ts` and the interceptors in `lib/axios.ts` are the most
 * breakage-prone code in the application — a rotating refresh chain, a
 * sessionStorage-backed JWT pair, a 401 that replays once and a 403 that
 * redirects — and nothing exercised any of it through a browser. Every other
 * spec depends on it and none asserts it.
 *
 * Read-only, and deliberately without a sign-out test. Signing out ends the
 * persona's Logto SSO session, not just this tab's: the saved `storageState`
 * every other spec boots from would be dead, and which spec noticed first would
 * depend on worker scheduling. Covering it needs a persona of its own, or a
 * project that runs last.
 */

import { test, expect, type Page } from '@playwright/test'
import { owner, persona, storageStatePath } from '../fixtures/personas'
import { apiResponse, openAs } from '../fixtures/auth'
import { t } from '../fixtures/i18n'

/** Where the sign-in flow shows up in a navigation trail. */
function signInDetours(visited: string[]): string[] {
  return visited.filter((url) => url.includes('/sign-in') || url.endsWith('/login'))
}

function recordNavigations(page: Page): string[] {
  const visited: string[] = []

  page.on('framenavigated', (frame) => {
    if (frame === page.mainFrame()) {
      visited.push(frame.url())
    }
  })
  return visited
}

test.describe('a live session', () => {
  test.use({ storageState: storageStatePath(owner.key) })

  test('survives a reload without going back to Logto', async ({ page }) => {
    await openAs(page, '/systems', /\/api\/systems(\?|$)/)

    const visited = recordNavigations(page)
    const exchange = apiResponse(page, /\/auth\/exchange/)

    await page.reload()
    expect((await exchange).status()).toBe(200)

    await expect(page).toHaveURL(/\/systems$/)
    await expect(page.getByRole('heading', { name: t('systems.title') })).toBeVisible()
    expect(
      signInDetours(visited),
      'a reload should re-exchange in place, not re-enter the sign-in flow',
    ).toEqual([])
  })

  test('re-mints the JWT pair when the tab has none', async ({ page }) => {
    await openAs(page, '/systems', /\/api\/systems(\?|$)/)

    // The pair lives in sessionStorage, which `storageState` does not capture,
    // so this is the state every spec actually starts in — asserted here rather
    // than merely relied upon. The Logto session in localStorage is what makes
    // the re-mint silent.
    await page.evaluate(() => {
      for (const key of Object.keys(sessionStorage)) {
        if (key.startsWith('my_')) {
          sessionStorage.removeItem(key)
        }
      }
    })

    const visited = recordNavigations(page)
    const exchange = apiResponse(page, /\/auth\/exchange/)

    await page.reload()
    expect((await exchange).status()).toBe(200)

    await expect
      .poll(() => page.evaluate(() => sessionStorage.getItem('my_jwt')), { timeout: 30_000 })
      .not.toBeNull()
    await expect(page.getByRole('heading', { name: t('systems.title') })).toBeVisible()
    expect(signInDetours(visited), 'the re-mint should need no redirect').toEqual([])
  })
})

/**
 * A persona that is refused something, so the 403 path can be driven. A customer
 * organization holds no hierarchy permission at all, which makes /distributors a
 * reliable refusal — see `rbac.spec.ts` for the full matrix.
 */
test.describe('a refusal mid-session', () => {
  const subject = persona('authz-d1r1c1-admin')

  test.use({ storageState: storageStatePath(subject.key) })

  test('offers a way back from the forbidden page', async ({ page }) => {
    await openAs(page, '/distributors')
    await expect(page).toHaveURL(/\/forbidden$/, { timeout: 30_000 })
    await expect(page.getByText(t('forbidden.title'))).toBeVisible()

    await page
      .getByRole('button', { name: t('common.go_to_page', { page: t('dashboard.title') }) })
      .click()

    await expect(page).toHaveURL(/\/dashboard$/, { timeout: 30_000 })
  })

  test('leaves the rest of the session usable', async ({ page }) => {
    await openAs(page, '/distributors')
    await expect(page).toHaveURL(/\/forbidden$/, { timeout: 30_000 })

    // The 403 interceptor redirects; it must not also tear the session down.
    // Getting that wrong turns one missing permission into a logout, which is
    // the kind of thing an API-level suite cannot see at all.
    const listed = apiResponse(page, /\/api\/systems(\?|$)/)
    await page.getByRole('navigation').locator('a[href="/systems"]').click()
    expect((await listed).status()).toBe(200)

    await expect(page.getByRole('heading', { name: t('systems.title') })).toBeVisible()
    expect(await page.evaluate(() => sessionStorage.getItem('my_jwt'))).not.toBeNull()
  })
})
