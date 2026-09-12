//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { defineConfig } from '@playwright/test'
import { sharedConfig, sharedUse } from './e2e/playwright.shared'

/**
 * Post-deploy smoke tests against a deployed environment, kept in a
 * configuration of their own: no `webServer` (the target is already serving)
 * and no `setup` dependency (that project signs in against the local fixture's
 * Logto application, while these authenticate as the dedicated account of the
 * environment under test).
 *
 * Read-only by construction. The target shares a database and a Logto tenant
 * with real users, which is the other half of why it is separated from
 * `playwright.config.ts`: no `E2E_*` variable can point the mutating specs
 * here by accident.
 *
 *   npm run test:e2e:smoke
 */

/**
 * The deployed environment under test. QA by default; set
 * E2E_SMOKE_BASE_URL to aim elsewhere — another deployed environment, or the
 * compose stack behind the proxy. It must serve both the application and
 * `/backend/api`, which a bare Vite dev server does not.
 */
const BASE_URL = process.env.E2E_SMOKE_BASE_URL ?? 'https://qa.my.nethesis.it'

export default defineConfig({
  ...sharedConfig,

  testDir: './e2e/smoke',

  use: {
    ...sharedUse,
    baseURL: BASE_URL,
  },

  projects: [{ name: 'smoke' }],
})
