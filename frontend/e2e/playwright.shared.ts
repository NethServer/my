//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { devices, type PlaywrightTestConfig } from '@playwright/test'
import { join } from 'node:path'

/**
 * Settings both Playwright configurations share — `playwright.config.ts` for
 * the local suite, `playwright.config.smoke.ts` for the deployed one. What
 * differs between them is only the target and who serves it.
 */

/**
 * Local convenience: `frontend/.env.e2e` (gitignored) holds what a developer
 * would otherwise retype on every run — the smoke credentials, and an
 * `E2E_SMOKE_BASE_URL` if the target is not the default QA. Missing is the
 * normal case, in CI included, where the values come from the environment.
 */
try {
  // Resolved against this file rather than the cwd, so the lookup does not
  // depend on which directory the run was launched from.
  process.loadEnvFile(join(import.meta.dirname, '..', '.env.e2e'))
} catch {
  // no such file: every value falls back to the environment
}

export const sharedConfig = {
  /**
   * Personas are shared state: several workers driving one persona's saved
   * session also share that persona's Logto refresh token, and rotating it in
   * parallel looks like token theft. Keep the worker count low until the
   * suite is big enough to justify partitioning personas across projects.
   */
  workers: 2,
  fullyParallel: false,

  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,

  reporter: process.env.CI
    ? [['github'], ['html', { open: 'never' }], ['list']]
    : [['html', { open: 'never' }], ['list']],
} satisfies PlaywrightTestConfig

export const sharedUse = {
  ...devices['Desktop Chrome'],

  /**
   * Pinned, and load-bearing. Selectors are resolved by translation key
   * through `e2e/fixtures/i18n.ts`, which reads the English catalogue only,
   * so a browser negotiating `it` would turn every `getByLabel` into a silent
   * miss. `auth.setup.ts` additionally clears the stored `preferences-*`
   * entry, because a locale saved there outranks the browser's.
   */
  locale: 'en-US',

  screenshot: 'only-on-failure',
  video: 'retain-on-failure',
  // retain-on-failure rather than on-first-retry: a trace on the *first*
  // failure is the point, otherwise a run with retries disabled produces none
  trace: 'retain-on-failure',
} satisfies PlaywrightTestConfig['use']
