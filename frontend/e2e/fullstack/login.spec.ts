//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Proves the auth mechanic the whole suite rests on: a session saved by the
 * setup project boots the application, survives the token exchange, and lands
 * on an authenticated page.
 */

import { test, expect } from '@playwright/test'
import { owner, storageStatePath } from '../fixtures/personas'

test.describe('a saved session', () => {
  test.use({ storageState: storageStatePath(owner.key) })

  test('lands on the dashboard without a Logto round trip', async ({ page }) => {
    // Record where the main frame actually went. A final-URL assertion cannot
    // tell a direct load from a bounce out to /login (or to Logto) and back,
    // and that round trip is half of what this test is about.
    const visited: string[] = []

    page.on('framenavigated', (frame) => {
      if (frame === page.mainFrame()) {
        visited.push(frame.url())
      }
    })

    await page.goto('/dashboard')
    await expect(page).toHaveURL(/\/dashboard$/)

    // The exchange ran on boot: this is the app's own JWT, not the Logto token.
    await expect
      .poll(() => page.evaluate(() => sessionStorage.getItem('my_jwt')), { timeout: 30_000 })
      .not.toBeNull()

    // The shell only renders for an authenticated user.
    await expect(page.getByRole('navigation')).toBeVisible()

    const detours = visited.filter((url) => url.includes('/sign-in') || url.endsWith('/login'))
    expect(detours, 'the saved session should boot without re-entering the sign-in flow').toEqual(
      [],
    )
  })
})

test.describe('no session', () => {
  // An empty state rather than a hand-built context, so the page still gets
  // every option from `use` in playwright.config.ts — baseURL and the pinned
  // locale included. A context made with browser.newContext() would not.
  test.use({ storageState: { cookies: [], origins: [] } })

  test('the guard sends an unauthenticated visitor to Logto', async ({ page }) => {
    await page.goto('/systems')
    await page.waitForURL(/\/sign-in/, { timeout: 60_000 })
    await expect(page.locator('input[name="identifier"]')).toBeVisible()
  })
})
