//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { defineConfig } from '@playwright/test'
import { sharedConfig, sharedUse } from './e2e/playwright.shared'

/**
 * The local suite: personas signed in once by `setup`, then the mutating
 * `fullstack` specs. Read-only checks against a deployed environment live in
 * `playwright.config.smoke.ts` instead — that separation is what keeps a
 * deployed origin out of reach of specs that create and delete organizations.
 */

/**
 * Fixed, not configurable. The Logto application the fixture was provisioned
 * against only accepts this origin as a sign-in redirect URI, so the OIDC
 * callback fails anywhere else — which is also why `docker-compose.e2e.yml`
 * publishes the proxy on 5173 in CI.
 */
const BASE_URL = 'http://localhost:5173'

export default defineConfig({
  ...sharedConfig,

  testDir: './e2e',

  use: {
    ...sharedUse,
    baseURL: BASE_URL,
  },

  projects: [
    /**
     * Signs each persona in through the real Logto form once and saves the
     * result, so the specs start authenticated. See e2e/setup/auth.setup.ts.
     */
    {
      name: 'setup',
      testMatch: /.*\.setup\.ts/,
    },

    /** Mutating specs. Require a local backend and a provisioned fixture. */
    {
      name: 'fullstack',
      testMatch: /fullstack\/.*\.spec\.ts/,
      dependencies: ['setup'],
    },
  ],

  /**
   * `reuseExistingServer` is what lets one config serve both ways of getting
   * an application onto 5173: locally Playwright starts the dev server itself,
   * while in CI the compose stack is already published there and Playwright
   * finds it and starts nothing.
   *
   * The catch is local: a plain `npm run dev` is reused as readily as
   * `npm run dev:e2e`, and it does not carry VITE_E2E — which is what
   * suppresses query auto-refetch and the Pinia Colada devtools panel, both of
   * which race assertions. Prefer `npm run dev:e2e` when running one by hand.
   */
  webServer: {
    command: 'npm run dev:e2e',
    url: BASE_URL,
    reuseExistingServer: true,
    timeout: 120_000,
  },
})
