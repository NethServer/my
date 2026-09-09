//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { defineConfig, devices } from '@playwright/test'

/**
 * Where the suite points. Unset means the local Vite dev server, which
 * `webServer` below starts (or reuses). Set it to run against a deployed
 * environment, e.g. E2E_BASE_URL=https://qa.my.nethesis.it for the smoke
 * project.
 *
 * The default must stay on port 5173: the Logto application the fixture was
 * provisioned against only accepts that origin as a sign-in redirect URI, so
 * the OIDC callback fails on any other port.
 */
const EXTERNAL_BASE_URL = process.env.E2E_BASE_URL
const BASE_URL = EXTERNAL_BASE_URL ?? 'http://localhost:5173'

export default defineConfig({
  testDir: './e2e',

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

  use: {
    baseURL: BASE_URL,

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
  },

  projects: [
    /**
     * Signs each persona in through the real Logto form once and saves the
     * result, so the specs start authenticated. See e2e/setup/auth.setup.ts.
     */
    {
      name: 'setup',
      testMatch: /.*\.setup\.ts/,
      use: { ...devices['Desktop Chrome'] },
    },

    /** Mutating specs. Require a local backend and a provisioned fixture. */
    {
      name: 'fullstack',
      testMatch: /fullstack\/.*\.spec\.ts/,
      dependencies: ['setup'],
      use: { ...devices['Desktop Chrome'] },
    },

    /**
     * Read-only specs against a deployed environment. Never mutating: they run
     * against a shared database and a shared Logto tenant.
     */
    {
      name: 'smoke',
      testMatch: /smoke\/.*\.spec\.ts/,
      // No dependency on `setup`: that project signs in against the local
      // fixture's Logto application. Smoke authenticates as the dedicated QA
      // e2e user instead, wired up alongside the first smoke spec.
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  /**
   * Only meaningful for the local target; a deployed environment is already
   * serving. `reuseExistingServer` keeps an already-running `npm run dev` from
   * being duplicated — note such a server does not carry VITE_E2E, so prefer
   * `npm run dev:e2e` when one is running by hand.
   */
  webServer: EXTERNAL_BASE_URL
    ? undefined
    : {
        command: 'npm run dev:e2e',
        url: BASE_URL,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
      },
})
