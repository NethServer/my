//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Proves the auth mechanic the whole suite rests on: a session saved by the
 * setup project boots the application, survives the token exchange, and lands
 * on an authenticated page.
 */

import { test, expect } from '@playwright/test'
import { owner, storageStatePath } from '../fixtures/personas'

test.use({ storageState: storageStatePath(owner.key) })

test('a saved session lands on the dashboard without a Logto round trip', async ({ page }) => {
  await page.goto('/dashboard')

  // Never bounced to /login, and never sent back out to the Logto origin.
  await expect(page).toHaveURL(/\/dashboard$/)

  // The exchange ran on boot: this is the app's own JWT, not the Logto token.
  await expect
    .poll(() => page.evaluate(() => sessionStorage.getItem('my_jwt')), { timeout: 30_000 })
    .not.toBeNull()

  // The shell only renders for an authenticated user.
  await expect(page.getByRole('navigation')).toBeVisible()
})

test('the guard sends an unauthenticated visitor to Logto', async ({ browser }) => {
  const context = await browser.newContext({ storageState: undefined })
  const page = await context.newPage()

  await page.goto('/systems')
  await page.waitForURL(/\/sign-in/, { timeout: 60_000 })
  await expect(page.locator('input[name="identifier"]')).toBeVisible()

  await context.close()
})
