//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * The one way a spec types a password: through the real Logto sign-in form,
 * without the password reaching the test output.
 *
 * `locator.fill` titles its step `Fill "<value>"`, so the password would land
 * in the HTML report, for passing tests too. `evaluate` is titled `Evaluate`
 * and renders no arguments. Traces would still record it — in the call
 * arguments, the DOM snapshots and the sign-in request body — which is why the
 * projects that sign in run with traces off (`playwright.config*.ts`).
 *
 * A new spec that needs a credential of any kind goes through here, and the
 * credential goes on the list `e2e/check-no-passwords.sh` greps for.
 */

import { test, type Page } from '@playwright/test'

export const REDACTED = '[REDACTED]'

const SIGN_IN_TIMEOUT = 60_000

/**
 * Sets an input's value the way React notices: its value tracker ignores an
 * assignment through the element's own property, so go through the prototype
 * setter and announce it. Without `announce`, React keeps what it had — which
 * is how the failure path swaps the value on screen and nothing else.
 */
function setValue(el: object, [value, announce]: [string, boolean]) {
  // The prototype is HTMLInputElement's, named indirectly because the suite is
  // type-checked without the DOM library.
  Object.getOwnPropertyDescriptor(Object.getPrototypeOf(el), 'value')!.set!.call(el, value)
  if (announce) {
    ;(el as EventTarget).dispatchEvent(new Event('input', { bubbles: true }))
  }
}

/**
 * Signs in from the application's root and returns once back on `/dashboard`.
 *
 * On failure the password field is left holding `[REDACTED]` before the error
 * propagates: the failure screenshot and the page snapshot in error-context.md
 * are both taken at teardown, and this way they show a field that was filled,
 * rather than the password or an empty field that reads as a step never run.
 */
export async function signIn(page: Page, email: string, password: string) {
  // The catch only runs if the helper's own waits expire before the test's
  // timeout does, which by default (30s) is shorter than either of them.
  test.info().setTimeout(test.info().timeout + 3 * SIGN_IN_TIMEOUT)

  // "/" redirects to /dashboard, the router guard bounces an unauthenticated
  // visitor to /login, and LoginView immediately calls signIn() — which is a
  // full-page navigation to the Logto-hosted form on another origin.
  await page.goto('/')
  await page.waitForURL(/\/sign-in/, { timeout: SIGN_IN_TIMEOUT })

  const passwordField = page.locator('input[name="password"]')
  try {
    await page.locator('input[name="identifier"]').fill(email)
    await passwordField.evaluate(setValue, [password, true] as [string, boolean])
    await page.locator('button[type="submit"]').click()

    // Back on our origin: /login-redirect completes the OIDC callback and
    // pushes to the saved deep link, or the dashboard.
    await page.waitForURL((url) => url.pathname === '/dashboard', { timeout: SIGN_IN_TIMEOUT })
  } catch (error) {
    await passwordField
      .evaluate(setValue, [REDACTED, false] as [string, boolean], { timeout: 1_000 })
      .catch(() => {})
    throw error
  }
}
