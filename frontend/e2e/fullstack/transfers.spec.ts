//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Changing the manager of customers and resellers through the interface.
 *
 * The hierarchy these specs move around is created over the API under the
 * reserved `e2e-` prefix and destroyed afterwards: two distributors, two
 * resellers under the first one and three customers under the first reseller.
 * The fixture organizations are never moved.
 */

import { test, expect, type Locator, type Page } from '@playwright/test'
import { owner, persona, storageStatePath } from '../fixtures/personas'
import { openAs } from '../fixtures/auth'
import { apiGet, apiPost } from '../fixtures/api'
import { t } from '../fixtures/i18n'
import {
  destroyE2eOrganization,
  e2eOrgName,
  e2eVat,
  listE2eOrganizations,
  type OrgType,
  type Organization,
} from '../fixtures/organizations'

type WithParent = Organization & { parent?: { logto_id: string; name: string } }

const created: string[] = []

let distributor: Organization
let otherDistributor: Organization
let reseller: Organization
let otherReseller: Organization
let customers: Organization[] = []
/** Shared by the three customers and nothing else, to filter the table on. */
let customerTag: string

async function createOrg(type: OrgType, name: string, parent?: Organization) {
  created.push(name)
  return apiPost<Organization>(`/${type}`, {
    name,
    custom_data: { vat: e2eVat() },
    ...(parent ? { created_by_organization_id: parent.logto_id } : {}),
  })
}

test.beforeAll(async () => {
  distributor = await createOrg('distributors', e2eOrgName('tf-dist'))
  otherDistributor = await createOrg('distributors', e2eOrgName('tf-dist'))
  reseller = await createOrg('resellers', e2eOrgName('tf-res'), distributor)
  otherReseller = await createOrg('resellers', e2eOrgName('tf-res'), distributor)

  customerTag = e2eOrgName('tf-cust')
  customers = []
  for (const suffix of ['a', 'b', 'c']) {
    customers.push(await createOrg('customers', `${customerTag}-${suffix}`, reseller))
  }
})

test.afterAll(async () => {
  for (const type of ['distributors', 'resellers', 'customers'] as OrgType[]) {
    for (const org of await listE2eOrganizations(type)) {
      if (created.includes(org.name)) {
        await destroyE2eOrganization(type, org)
      }
    }
  }
})

function rowOf(page: Page, name: string): Locator {
  return page.getByRole('row').filter({ hasText: name })
}

/** The open change-manager drawer, identified by the field only it has. */
function drawerOf(page: Page): Locator {
  return page.locator('form').filter({ has: page.getByLabel(t('customers.new_manager')) })
}

/** The field searches the backend as you type: the option exists once it answers. */
async function chooseManager(page: Page, drawer: Locator, name: string) {
  const field = drawer.getByRole('combobox')
  await field.click()
  await field.fill(name)
  const option = page.getByRole('option', { name: new RegExp(`^${name}`) })
  await expect(option).toBeVisible({ timeout: 30_000 })
  await option.click()
}

test.describe('as the owner', () => {
  test.use({ storageState: storageStatePath(owner.key) })

  test('moves several customers under another reseller at once', async ({ page }) => {
    await openAs(page, '/customers', /\/api\/customers/)

    await page.getByPlaceholder(t('customers.filter_customers')).fill(customerTag)
    for (const customer of customers) {
      await expect(rowOf(page, customer.name)).toBeVisible({ timeout: 30_000 })
      await expect(rowOf(page, customer.name)).toContainText(reseller.name)
    }

    await page.getByRole('checkbox', { name: t('customers.select_all_customers') }).check()
    await page.getByRole('button', { name: t('customers.change_manager') }).click()

    const drawer = drawerOf(page)
    await expect(drawer).toBeVisible()
    await chooseManager(page, drawer, otherReseller.name)

    // the preview pass ran for every customer before the move can be confirmed
    const confirm = drawer.getByRole('checkbox', {
      name: t('customers.bulk_confirm_label', { name: otherReseller.name }),
    })
    await expect(confirm).toBeVisible({ timeout: 60_000 })
    await confirm.check()
    await drawer.getByRole('button', { name: t('customers.change_manager') }).click()

    const close = drawer.getByRole('button', { name: t('common.close') })
    await expect(close).toBeVisible({ timeout: 60_000 })
    await close.click()
    await expect(drawer).toBeHidden()

    // really moved in the backend, not only on screen
    for (const customer of customers) {
      const stored = await apiGet<WithParent>(`/customers/${customer.logto_id}`)
      expect(stored.parent?.logto_id, `${customer.name} should be under the new reseller`).toBe(
        otherReseller.logto_id,
      )
      await expect(rowOf(page, customer.name)).toContainText(otherReseller.name, {
        timeout: 30_000,
      })
    }
  })

  test('refuses to move customers already managed by the target', async ({ page }) => {
    // runs after the bulk move: the three customers now sit under otherReseller
    await openAs(page, '/customers', /\/api\/customers/)

    await page.getByPlaceholder(t('customers.filter_customers')).fill(customerTag)
    await expect(rowOf(page, customers[0].name)).toBeVisible({ timeout: 30_000 })
    await page.getByRole('checkbox', { name: t('customers.select_all_customers') }).check()
    await page.getByRole('button', { name: t('customers.change_manager') }).click()

    const drawer = drawerOf(page)
    await chooseManager(page, drawer, otherReseller.name)

    await expect(drawer.getByText(t('customers.transfer_same_organization')).first()).toBeVisible({
      timeout: 60_000,
    })
    await expect(drawer.getByRole('button', { name: t('customers.change_manager') })).toBeDisabled()
  })

  test('moves a reseller under another distributor', async ({ page }) => {
    await openAs(page, '/resellers', /\/api\/resellers/)

    await page.getByPlaceholder(t('resellers.filter_resellers')).fill(otherReseller.name)
    const row = rowOf(page, otherReseller.name)
    await expect(row).toBeVisible({ timeout: 30_000 })
    await expect(row).toContainText(distributor.name)

    await row.getByRole('button', { name: /menu/i }).click()
    await page.getByRole('menuitem', { name: t('resellers.change_manager') }).click()

    const drawer = drawerOf(page)
    await expect(drawer).toBeVisible()
    await chooseManager(page, drawer, otherDistributor.name)

    const confirm = drawer.getByRole('checkbox', {
      name: t('resellers.transfer_confirm_label', { name: otherDistributor.name }),
    })
    await expect(confirm).toBeVisible({ timeout: 30_000 })
    await confirm.check()
    await drawer.getByRole('button', { name: t('resellers.change_manager') }).click()
    await expect(drawer).toBeHidden({ timeout: 30_000 })

    const stored = await apiGet<WithParent>(`/resellers/${otherReseller.logto_id}`)
    expect(stored.parent?.logto_id).toBe(otherDistributor.logto_id)
    await expect(rowOf(page, otherReseller.name)).toContainText(otherDistributor.name, {
      timeout: 30_000,
    })
  })
})

test.describe('as a reseller admin', () => {
  test.use({ storageState: storageStatePath(persona('authz-d1r1-admin').key) })

  test('offers no bulk change of manager', async ({ page }) => {
    await openAs(page, '/customers', /\/api\/customers/)

    await expect(page.getByRole('row').nth(1)).toBeVisible({ timeout: 30_000 })
    await expect(
      page.getByRole('checkbox', { name: t('customers.select_all_customers') }),
    ).toHaveCount(0)
  })
})
