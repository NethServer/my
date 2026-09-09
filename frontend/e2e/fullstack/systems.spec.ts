//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * The systems list and detail views, and the registration handshake.
 *
 * Registration is the one flow here that has no interface of its own: an
 * appliance completes it by posting its secret to the public
 * `POST /systems/register`. What this asserts is that the handshake is
 * *reflected* in the interface — a system that has not registered is offered
 * "Regenerate secret" (`SystemsTable.vue:297`) and a registered one is not,
 * because regenerating a secret an appliance is already using would lock it
 * out.
 */

import { test, expect, type Page } from '@playwright/test'
import { owner, persona, storageStatePath } from '../fixtures/personas'
import { apiResponse, openAs } from '../fixtures/auth'
import { t } from '../fixtures/i18n'
import {
  createE2eSystem,
  destroyE2eSystem,
  getSystem,
  listE2eSystems,
  registerE2eSystem,
  sweepE2eSystems,
  type System,
} from '../fixtures/systems'

test.use({ storageState: storageStatePath(owner.key) })

/** A customer organization from the authz fixture to hang systems off. */
const CUSTOMER_ORG = persona('authz-d1r1c1-admin')

const created: string[] = []

test.beforeAll(async () => {
  await sweepE2eSystems()
})

test.afterAll(async () => {
  for (const system of await listE2eSystems()) {
    if (created.includes(system.name)) {
      await destroyE2eSystem(system)
    }
  }
})

async function newSystem(): Promise<System> {
  const system = await createE2eSystem(CUSTOMER_ORG.orgId)
  created.push(system.name)
  return system
}

/** Open the systems list and wait for the rows to arrive. */
async function openSystems(page: Page) {
  await openAs(page, '/systems', /\/api\/systems(\?|$)/)
}

/** Narrow the list to one system, so its row is unambiguous. */
async function filterTo(page: Page, name: string) {
  await page.getByPlaceholder(t('systems.filter_systems')).fill(name)
  await expect(page.getByRole('row').filter({ hasText: name })).toBeVisible({ timeout: 30_000 })
}

test('lists a newly created system and filters down to it', async ({ page }) => {
  const system = await newSystem()
  const other = await newSystem()

  await openSystems(page)
  await filterTo(page, system.name)

  // The filter is doing real work: the sibling is gone from the list.
  await expect(page.getByRole('row').filter({ hasText: other.name })).toHaveCount(0)
})

test('opens the detail view for a system', async ({ page }) => {
  const system = await newSystem()

  await openSystems(page)
  await filterTo(page, system.name)

  // The row's own affordance, not a click anywhere on the row.
  await page
    .getByRole('row')
    .filter({ hasText: system.name })
    .getByRole('button', { name: t('common.details') })
    .click()

  await expect(page).toHaveURL(new RegExp(`/systems/${system.id}$`), { timeout: 30_000 })
  await expect(page.getByText(system.name).first()).toBeVisible()
})

test('stops offering to regenerate the secret once the system registers', async ({ page }) => {
  const system = await newSystem()

  // Precondition: created but not registered.
  expect(system.registered_at ?? null).toBeNull()

  await openSystems(page)
  await filterTo(page, system.name)

  const row = () => page.getByRole('row').filter({ hasText: system.name })
  await row().getByRole('button', { name: /menu/i }).click()
  await expect(page.getByRole('menuitem', { name: t('systems.regenerate_secret') })).toBeVisible()
  await page.keyboard.press('Escape')

  // The appliance side of registration: the public endpoint, no token.
  const registration = await registerE2eSystem(system.system_secret)
  expect(registration.system_key).toBe(system.system_key)
  expect((await getSystem(system.id)).registered_at ?? null).not.toBeNull()

  const listed = apiResponse(page, /\/api\/systems(\?|$)/)
  await page.reload()
  await listed
  await filterTo(page, system.name)

  await row().getByRole('button', { name: /menu/i }).click()
  await expect(page.getByRole('menuitem', { name: t('systems.regenerate_secret') })).toHaveCount(0)
})
