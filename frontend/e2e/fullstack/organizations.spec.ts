//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Creating and editing organizations through the interface.
 *
 * These are the suite's only writing specs, and they write to a real Logto
 * tenant: creating a distributor calls the Logto Management API, which is why
 * this cannot be exercised against a stubbed backend. Everything they create is
 * named under the reserved `e2e-` prefix and destroyed afterwards, and the
 * teardown helper refuses any name outside that prefix. See
 * `e2e/fixtures/organizations.ts`.
 */

import { test, expect, type Locator, type Page } from '@playwright/test'
import { owner, storageStatePath } from '../fixtures/personas'
import { openAs } from '../fixtures/auth'
import { t } from '../fixtures/i18n'
import {
  destroyE2eOrganization,
  e2eOrgName,
  e2eVat,
  listE2eOrganizations,
  sweepE2eOrganizations,
} from '../fixtures/organizations'

// The owner can manage every tier, so one session covers the whole hierarchy.
test.use({ storageState: storageStatePath(owner.key) })

/** Names created here, destroyed in afterAll even if an assertion failed. */
const created: string[] = []

test.beforeAll(async () => {
  // An earlier run that crashed before its teardown would otherwise leave rows
  // that make "the new distributor appears in the list" ambiguous.
  await sweepE2eOrganizations()
})

test.afterAll(async () => {
  for (const org of await listE2eOrganizations('distributors')) {
    if (created.includes(org.name)) {
      await destroyE2eOrganization('distributors', org)
    }
  }
})

/** The open create/edit drawer, identified by a field only it has. */
function drawerOf(page: Page): Locator {
  return page.locator('form').filter({ has: page.getByLabel(t('organizations.name')) })
}

async function openCreateDrawer(page: Page): Promise<Locator> {
  await page.getByRole('button', { name: t('distributors.create_distributor') }).click()

  const drawer = drawerOf(page)
  await expect(drawer).toBeVisible()
  return drawer
}

/** The table row for an organization, by name. */
function rowOf(page: Page, name: string): Locator {
  return page.getByRole('row').filter({ hasText: name })
}

test('creates a distributor and lists it', async ({ page }) => {
  await openAs(page, '/distributors')
  await page.waitForResponse((r) => r.url().includes('/api/distributors'))

  const name = e2eOrgName('dist')
  const vat = e2eVat()
  created.push(name)

  const drawer = await openCreateDrawer(page)
  await drawer.getByLabel(t('organizations.name')).fill(name)
  await drawer.getByLabel(t('organizations.vat_number')).fill(vat)
  await drawer.getByRole('button', { name: t('distributors.create_distributor') }).click()

  // The drawer closes on success, and the list query is invalidated.
  await expect(drawer).toBeHidden({ timeout: 30_000 })
  await expect(rowOf(page, name)).toBeVisible({ timeout: 30_000 })
  await expect(rowOf(page, name)).toContainText(vat)

  // It is really in the backend, not only on screen.
  const stored = (await listE2eOrganizations('distributors')).find((o) => o.name === name)
  expect(stored, `${name} should exist in the backend after creating it`).toBeDefined()
  expect(stored?.custom_data?.vat).toBe(vat)
})

test('refuses a distributor with no VAT number', async ({ page }) => {
  await openAs(page, '/distributors')
  await page.waitForResponse((r) => r.url().includes('/api/distributors'))

  const name = e2eOrgName('novat')
  const drawer = await openCreateDrawer(page)

  await drawer.getByLabel(t('organizations.name')).fill(name)
  // VAT deliberately left empty
  await drawer.getByRole('button', { name: t('distributors.create_distributor') }).click()

  await expect(drawer.getByText(t('organizations.custom_data_vat_cannot_be_empty'))).toBeVisible()

  // Rejected in the browser: nothing reached the backend.
  await expect(drawer).toBeVisible()
  const stored = (await listE2eOrganizations('distributors')).find((o) => o.name === name)
  expect(stored, 'a rejected form must not create anything').toBeUndefined()
})

test('edits a distributor and keeps the change', async ({ page }) => {
  await openAs(page, '/distributors')
  await page.waitForResponse((r) => r.url().includes('/api/distributors'))

  const name = e2eOrgName('edit')
  created.push(name)

  const createDrawer = await openCreateDrawer(page)
  await createDrawer.getByLabel(t('organizations.name')).fill(name)
  await createDrawer.getByLabel(t('organizations.vat_number')).fill(e2eVat())
  await createDrawer.getByRole('button', { name: t('distributors.create_distributor') }).click()
  await expect(createDrawer).toBeHidden({ timeout: 30_000 })

  const row = rowOf(page, name)
  await expect(row).toBeVisible({ timeout: 30_000 })

  await row.getByRole('button', { name: /menu/i }).click()
  await page.getByRole('menuitem', { name: t('common.edit') }).click()

  const editDrawer = drawerOf(page)
  await expect(editDrawer).toBeVisible()
  await expect(editDrawer.getByLabel(t('organizations.name'))).toHaveValue(name)

  const city = 'Lucca'
  await editDrawer.getByLabel(new RegExp(`^${t('organizations.city')}`)).fill(city)
  await editDrawer.getByRole('button', { name: t('distributors.save_distributor') }).click()
  await expect(editDrawer).toBeHidden({ timeout: 30_000 })

  // Reopen rather than trusting the row: this proves the value was persisted.
  await rowOf(page, name).getByRole('button', { name: /menu/i }).click()
  await page.getByRole('menuitem', { name: t('common.edit') }).click()

  const reopened = drawerOf(page)
  await expect(reopened.getByLabel(new RegExp(`^${t('organizations.city')}`))).toHaveValue(city)
})
