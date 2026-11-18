//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Changing the manager of customers and resellers through the interface.
 *
 * The hierarchy these specs move around is created over the API under the
 * reserved `e2e-` prefix and destroyed afterwards: two distributors, two
 * resellers under the first one and three customers under the first reseller.
 * The fixture organizations are never moved; the visibility tests move e2e
 * companies under them, so the admins of the fixture managers can check what
 * the move gives and takes away.
 */

import { test, expect, type Locator, type Page } from '@playwright/test'
import { fixtureOrg, owner, persona, storageStatePath } from '../fixtures/personas'
import { apiResponse, openAs } from '../fixtures/auth'
import { apiGet, apiGetAs, apiPost } from '../fixtures/api'
import { t } from '../fixtures/i18n'
import {
  destroyE2eOrganization,
  e2eOrgName,
  e2eVat,
  listE2eOrganizations,
  type OrgType,
  type Organization,
} from '../fixtures/organizations'
import {
  createE2eSystem,
  destroyE2eSystem,
  registerE2eSystem,
  type System,
} from '../fixtures/systems'
import {
  destroyE2eUser,
  e2eUserEmail,
  e2eUserName,
  roleIdByName,
  type User,
} from '../fixtures/users'

type WithParent = Organization & { parent?: { logto_id: string; name: string } }

const created: string[] = []
const createdSystems: System[] = []
const createdUsers: User[] = []

let distributor: Organization
let otherDistributor: Organization
let reseller: Organization
let otherReseller: Organization
let customers: Organization[] = []
/** Shared by the three customers and nothing else, to filter the table on. */
let customerTag: string
let movedSystem: System | undefined

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
  if (movedSystem) {
    await destroyE2eSystem(movedSystem)
  }
  for (const system of createdSystems) {
    await destroyE2eSystem(system)
  }
  for (const user of createdUsers) {
    await destroyE2eUser(user)
  }
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

/**
 * The open change-manager drawer: the form inside the dialog with that title.
 * The bulk drawer hides the New manager field once the move is done, so the
 * field cannot identify it; and the dialog element itself has no size, so it
 * would always read as hidden. Customers and resellers share the title text.
 */
function drawerOf(page: Page): Locator {
  return page.getByRole('dialog', { name: t('customers.change_manager') }).locator('form')
}

/**
 * Searches the customers list and waits for the list to answer. The bulk change
 * of manager takes the search the table applied, which follows the typing after
 * a pause: the rows can already be on the unfiltered first page, visible before
 * the search runs.
 */
async function searchCustomers(page: Page, text: string) {
  const searched = apiResponse(page, new RegExp(`/api/customers\\?.*search=${text}`))
  await page.getByPlaceholder(t('customers.filter_customers')).fill(text)
  await searched
}

/** Opens the bulk change of manager from the Actions menu of the page header. */
async function openBulkChangeManager(page: Page, entity: 'customers' | 'resellers') {
  await page.getByRole('button', { name: t('common.actions'), exact: true }).click()
  await page.getByRole('menuitem', { name: t(`${entity}.change_manager`) }).click()
}

/** The plural form of a `none | one | many` message, filled for `count`. */
function plural(key: string, count: number, params: Record<string, string> = {}): string {
  const forms = t(key, { count: String(count), ...params }).split(' | ')
  return forms[Math.min(count, forms.length - 1)]
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

/** Something moved along with a company, and where a manager would find it. */
type Reachable = {
  label: string
  /** The detail endpoint: 200 when the manager may open it. */
  detail: string
  /** A search that lists it, and the key of the list in the response. */
  list: string
  listKey: 'customers' | 'resellers' | 'systems' | 'users'
  id: string
}

/**
 * What a fixture manager can reach, through the detail endpoint and through a
 * list, as its admin: the backend scopes both, along different code paths, so
 * a move must show up in each.
 */
async function expectReach(personaKey: string, objects: Reachable[], visible: boolean) {
  const verb = visible ? 'should' : 'should not'
  for (const object of objects) {
    const detail = await apiGetAs(persona(personaKey).key, object.detail)
    expect(detail.status === 200, `${personaKey} ${verb} open ${object.label}`).toBe(visible)

    const list = await apiGetAs<Record<string, { id?: string; logto_id?: string }[]>>(
      personaKey,
      object.list,
    )
    const listed = (list.data?.[object.listKey] ?? []).some(
      (row) => row.id === object.id || row.logto_id === object.id,
    )
    expect(listed, `${personaKey} ${verb} list ${object.label}`).toBe(visible)
  }
}

const asParent = (logtoId: string): Organization => ({ id: '', logto_id: logtoId, name: '' })

const search = (path: string, name: string) => `${path}?search=${encodeURIComponent(name)}`

test.describe('as the owner', () => {
  test.use({ storageState: storageStatePath(owner.key) })

  test('moves several customers under another reseller at once', async ({ page }) => {
    await openAs(page, '/customers', /\/api\/customers/)

    await searchCustomers(page, customerTag)
    for (const customer of customers) {
      await expect(rowOf(page, customer.name)).toBeVisible({ timeout: 30_000 })
      await expect(rowOf(page, customer.name)).toContainText(reseller.name)
    }

    await openBulkChangeManager(page, 'customers')

    // the move covers the filtered customers, all of them and nothing else
    const drawer = drawerOf(page)
    await expect(drawer).toBeVisible()
    await expect(drawer.getByText(plural('customers.bulk_to_move', customers.length))).toBeVisible({
      timeout: 30_000,
    })
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

    await searchCustomers(page, customerTag)
    await expect(rowOf(page, customers[0].name)).toBeVisible({ timeout: 30_000 })
    await openBulkChangeManager(page, 'customers')

    const drawer = drawerOf(page)
    await chooseManager(page, drawer, otherReseller.name)

    // the reasons sit in an expandable list, closed until asked for
    const skipped = drawer.getByRole('button', {
      name: plural('customers.bulk_skipped_title', customers.length),
    })
    await expect(skipped).toBeVisible({ timeout: 60_000 })
    await skipped.click()
    await expect(skipped).toHaveAttribute('aria-expanded', 'true')
    await expect(drawer.getByText(t('customers.transfer_same_organization')).first()).toBeVisible()
    // nothing to move: no confirmation to give, and the button moves nothing
    await expect(drawer.getByRole('checkbox')).toHaveCount(0)
    await drawer.getByRole('button', { name: t('customers.change_manager') }).click()
    await expect(drawer.getByText(t('customers.transfer_same_organization')).first()).toBeVisible()
  })

  test('refuses a bulk change of manager on the unfiltered list', async ({ page }) => {
    await openAs(page, '/customers', /\/api\/customers/)
    await expect(page.getByRole('row').nth(1)).toBeVisible({ timeout: 30_000 })

    await openBulkChangeManager(page, 'customers')

    const drawer = page
      .locator('form')
      .filter({ hasText: t('customers.bulk_change_manager_description') })
    await expect(drawer.getByText(t('customers.bulk_filter_required_title'))).toBeVisible()
    await expect(drawer.getByRole('combobox')).toHaveCount(0)
    await expect(drawer.getByRole('button', { name: t('customers.change_manager') })).toHaveCount(0)
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

  test('warns about the add-ons that stop renewing when a system changes company', async ({
    page,
  }) => {
    const [from, to] = customers
    movedSystem = await createE2eSystem(from.logto_id)
    await registerE2eSystem(movedSystem.system_secret)
    // no buyer on record: the renewal cannot follow the system
    const validUntil = new Date(Date.now() + 200 * 24 * 60 * 60 * 1000).toISOString()
    await apiPost(`/systems/${movedSystem.id}/entitlements`, {
      entitlement: 'nsec-ha',
      valid_until: validUntil,
    })

    await openAs(page, '/systems', /\/api\/systems(\?|$)/)
    await page.getByPlaceholder(t('systems.filter_systems')).fill(movedSystem.name)
    const row = rowOf(page, movedSystem.name)
    await expect(row).toBeVisible({ timeout: 30_000 })
    await row.getByRole('button', { name: /menu/i }).click()
    await page.getByRole('menuitem', { name: t('common.edit') }).click()

    const drawer = page.locator('form').filter({ has: page.getByLabel(t('systems.organization')) })
    await expect(drawer).toBeVisible()
    const warning = drawer.getByText(t('customers.addons_renewal_stop_title').split(' | ')[0])
    await expect(warning).toHaveCount(0)

    await chooseManager(page, drawer, to.name)
    await expect(warning).toBeVisible({ timeout: 30_000 })

    await drawer.getByRole('button', { name: t('common.save') }).click()
    await expect(drawer).toBeHidden({ timeout: 30_000 })

    const stored = await apiGet<{ organization: { logto_id: string } }>(
      `/systems/${movedSystem.id}`,
    )
    expect(stored.organization.logto_id).toBe(to.logto_id)
    const { entitlements } = await apiGet<{
      entitlements: { entitlement: string; renewal_stopped_reason?: string }[]
    }>(`/systems/${movedSystem.id}/entitlements`)
    expect(entitlements.find((g) => g.entitlement === 'nsec-ha')?.renewal_stopped_reason).toBe(
      'system_transferred',
    )
  })
})

test.describe('what the managers reach after a move', () => {
  test.use({ storageState: storageStatePath(owner.key) })

  test('a customer moved to another reseller takes its systems and users along', async ({
    page,
  }) => {
    const from = fixtureOrg('d1r1')
    const to = fixtureOrg('d1r2')
    const customer = await createOrg('customers', e2eOrgName('tf-vis'), asParent(from.logtoId))
    const system = await createE2eSystem(customer.logto_id)
    createdSystems.push(system)
    const user = await apiPost<User>('/users', {
      email: e2eUserEmail(),
      name: e2eUserName(),
      organization_id: customer.logto_id,
      user_role_ids: [await roleIdByName('Reader')],
    })
    createdUsers.push(user)

    const moved: Reachable[] = [
      {
        label: 'the customer',
        detail: `/customers/${customer.logto_id}`,
        list: search('/customers', customer.name),
        listKey: 'customers',
        id: customer.logto_id,
      },
      {
        label: 'its system',
        detail: `/systems/${system.id}`,
        list: search('/systems', system.name),
        listKey: 'systems',
        id: system.id,
      },
      {
        label: 'its user',
        detail: `/users/${user.logto_id}`,
        list: search('/users', user.name),
        listKey: 'users',
        id: user.logto_id,
      },
    ]
    await expectReach('authz-d1r1-admin', moved, true)
    await expectReach('authz-d1r2-admin', moved, false)

    await openAs(page, '/customers', /\/api\/customers/)
    await page.getByPlaceholder(t('customers.filter_customers')).fill(customer.name)
    const row = rowOf(page, customer.name)
    await expect(row).toBeVisible({ timeout: 30_000 })
    await row.getByRole('button', { name: /menu/i }).click()
    await page.getByRole('menuitem', { name: t('customers.change_manager') }).click()

    const drawer = drawerOf(page)
    await chooseManager(page, drawer, to.name)
    const confirm = drawer.getByRole('checkbox', {
      name: t('customers.transfer_confirm_label', { name: to.name }),
    })
    await expect(confirm).toBeVisible({ timeout: 30_000 })
    await confirm.check()
    await drawer.getByRole('button', { name: t('customers.change_manager') }).click()
    await expect(drawer).toBeHidden({ timeout: 30_000 })

    await expectReach('authz-d1r2-admin', moved, true)
    await expectReach('authz-d1r1-admin', moved, false)
  })

  test('a reseller moved to another distributor takes its customers and systems along', async ({
    page,
  }) => {
    const from = fixtureOrg('d1')
    const to = fixtureOrg('d2')
    const movedReseller = await createOrg('resellers', e2eOrgName('tf-vis'), asParent(from.logtoId))
    const customer = await createOrg('customers', e2eOrgName('tf-vis'), movedReseller)
    const system = await createE2eSystem(customer.logto_id)
    createdSystems.push(system)

    const moved: Reachable[] = [
      {
        label: 'the reseller',
        detail: `/resellers/${movedReseller.logto_id}`,
        list: search('/resellers', movedReseller.name),
        listKey: 'resellers',
        id: movedReseller.logto_id,
      },
      {
        label: "the reseller's customer",
        detail: `/customers/${customer.logto_id}`,
        list: search('/customers', customer.name),
        listKey: 'customers',
        id: customer.logto_id,
      },
      {
        label: "that customer's system",
        detail: `/systems/${system.id}`,
        list: search('/systems', system.name),
        listKey: 'systems',
        id: system.id,
      },
    ]
    await expectReach('authz-d1-admin', moved, true)
    await expectReach('authz-d2-admin', moved, false)

    await openAs(page, '/resellers', /\/api\/resellers/)
    await page.getByPlaceholder(t('resellers.filter_resellers')).fill(movedReseller.name)
    const row = rowOf(page, movedReseller.name)
    await expect(row).toBeVisible({ timeout: 30_000 })
    await row.getByRole('button', { name: /menu/i }).click()
    await page.getByRole('menuitem', { name: t('resellers.change_manager') }).click()

    const drawer = drawerOf(page)
    await chooseManager(page, drawer, to.name)
    const confirm = drawer.getByRole('checkbox', {
      name: t('resellers.transfer_confirm_label', { name: to.name }),
    })
    await expect(confirm).toBeVisible({ timeout: 30_000 })
    await confirm.check()
    await drawer.getByRole('button', { name: t('resellers.change_manager') }).click()
    await expect(drawer).toBeHidden({ timeout: 30_000 })

    await expectReach('authz-d2-admin', moved, true)
    await expectReach('authz-d1-admin', moved, false)
  })
})

test.describe('as a reseller admin', () => {
  test.use({ storageState: storageStatePath(persona('authz-d1r1-admin').key) })

  test('offers no bulk change of manager', async ({ page }) => {
    await openAs(page, '/customers', /\/api\/customers/)

    await expect(page.getByRole('row').nth(1)).toBeVisible({ timeout: 30_000 })
    await page.getByRole('button', { name: t('common.actions'), exact: true }).click()
    await expect(
      page.getByRole('menuitem', { name: t('customers.export_customers_to_csv') }),
    ).toBeVisible()
    await expect(page.getByRole('menuitem', { name: t('customers.change_manager') })).toHaveCount(0)
  })
})
