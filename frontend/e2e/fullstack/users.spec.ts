//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Creating, editing and archiving a user, and assigning technical roles.
 *
 * The lifecycle runs as one serial sequence rather than as independent tests,
 * for a reason that is not tidiness: creating a user makes the backend send a
 * welcome email with a temporary password
 * (`services/local/users.go:351`). Sharing one user across create, edit and
 * archive keeps that to a single message per run instead of one per test.
 *
 * Everything created is plus sub-addressed with the reserved tag and destroyed
 * afterwards; teardown refuses any address without it. See
 * `e2e/fixtures/users.ts`.
 */

import { test, expect, type Locator, type Page } from '@playwright/test'
import { owner, persona, storageStatePath } from '../fixtures/personas'
import { openAs } from '../fixtures/auth'
import { t } from '../fixtures/i18n'
import {
  destroyE2eUser,
  e2eUserEmail,
  e2eUserName,
  findE2eUser,
  listE2eUsers,
  roleIdByName,
  sweepE2eUsers,
} from '../fixtures/users'

test.use({ storageState: storageStatePath(owner.key) })

/** A customer organization from the authz fixture to hang users off. */
const ORG_NAME = persona('authz-d1r1c1-admin').orgName

const created: string[] = []

test.beforeAll(async () => {
  // Leftovers from a run that crashed before teardown would make "the new user
  // appears in the list" ambiguous.
  await sweepE2eUsers()
})

test.afterAll(async () => {
  for (const user of await listE2eUsers()) {
    if (created.includes(user.email)) {
      await destroyE2eUser(user)
    }
  }
})

/** The open create/edit drawer, identified by a field only it has. */
function drawerOf(page: Page): Locator {
  return page.locator('form').filter({ has: page.getByLabel(t('users.email'), { exact: true }) })
}

async function openUsers(page: Page) {
  await openAs(page, '/users')
  await page.waitForResponse((r) => /\/api\/users(\?|$)/.test(r.url()))
}

/**
 * Pick an organization. The field searches the backend as you type, so the
 * option only exists once that request has come back.
 */
async function chooseCompany(page: Page, name: string) {
  const company = page.getByRole('combobox').nth(0)
  await company.click()
  await company.fill(name)
  const option = page.getByRole('option', { name: new RegExp(`^${name}`) })
  await expect(option).toBeVisible({ timeout: 30_000 })
  await option.click()
}

/** Pick a technical role. Options render as the name followed by its description. */
async function chooseRole(page: Page, roleLabel: string) {
  await page.getByRole('combobox').nth(1).click()
  await page.getByRole('option', { name: new RegExp(`^${roleLabel}`) }).click()
}

function rowOf(page: Page, email: string): Locator {
  return page.getByRole('row').filter({ hasText: email })
}

/**
 * Narrow the list to one user. The listing is paginated and there are dozens of
 * fixture accounts, so a new user is rarely on the first page — without this
 * the assertions would depend on where it happened to sort.
 */
async function filterTo(page: Page, email: string) {
  await page.getByPlaceholder(t('users.filter_users')).fill(email)
  await expect(rowOf(page, email)).toBeVisible({ timeout: 30_000 })
}

test.describe.configure({ mode: 'serial' })

test.describe('user lifecycle', () => {
  const name = e2eUserName()
  const email = e2eUserEmail()

  test('creates a user with a role and lists it', async ({ page }) => {
    created.push(email)
    await openUsers(page)

    await page.getByRole('button', { name: t('users.create_user') }).click()
    const drawer = drawerOf(page)
    await expect(drawer).toBeVisible()

    await drawer.getByLabel(t('users.name'), { exact: true }).fill(name)
    await drawer.getByLabel(t('users.email'), { exact: true }).fill(email)
    await chooseCompany(page, ORG_NAME)
    await chooseRole(page, t('user_roles.reader'))
    await drawer.getByRole('button', { name: t('users.create_user') }).click()

    await expect(drawer).toBeHidden({ timeout: 30_000 })
    await filterTo(page, email)
    await expect(rowOf(page, email)).toContainText(t('user_roles.reader'))

    // The row is one thing; what the backend stored is another.
    const stored = await findE2eUser(email)
    expect(stored, `${email} should exist in the backend after creating it`).toBeDefined()
    expect(stored?.roles?.map((r) => r.id)).toEqual([await roleIdByName('Reader')])
    expect(stored?.organization?.name, 'the user should land in the chosen company').toBe(ORG_NAME)
  })

  test('changes the assigned role and keeps the change', async ({ page }) => {
    await openUsers(page)

    await filterTo(page, email)
    await rowOf(page, email)
      .getByRole('button', { name: t('common.edit') })
      .click()

    const drawer = drawerOf(page)
    await expect(drawer).toBeVisible()
    await expect(drawer.getByLabel(t('users.email'), { exact: true })).toHaveValue(email)

    await chooseRole(page, t('user_roles.support'))
    await drawer.getByRole('button', { name: t('users.save_user') }).click()
    await expect(drawer).toBeHidden({ timeout: 30_000 })

    // Both halves matter: the badge the operator sees, and the role the backend
    // will actually authorize against.
    await expect(rowOf(page, email)).toContainText(t('user_roles.support'), { timeout: 30_000 })
    expect((await findE2eUser(email))?.roles?.map((r) => r.id)).toEqual([
      await roleIdByName('Support'),
    ])
  })

  test('archives the user', async ({ page }) => {
    await openUsers(page)

    await filterTo(page, email)
    await rowOf(page, email).getByRole('button', { name: /menu/i }).click()
    await page.getByRole('menuitem', { name: t('common.archive') }).click()

    // The modal names the destructive action rather than just confirming.
    await page
      .getByRole('button', { name: t('common.archive') })
      .last()
      .click()

    // Archived users drop out of the default listing.
    await expect(rowOf(page, email)).toHaveCount(0, { timeout: 30_000 })
  })
})

test('refuses a user with no role assigned', async ({ page }) => {
  const name = e2eUserName()
  const email = e2eUserEmail()

  await openUsers(page)
  await page.getByRole('button', { name: t('users.create_user') }).click()

  const drawer = drawerOf(page)
  await drawer.getByLabel(t('users.name'), { exact: true }).fill(name)
  await drawer.getByLabel(t('users.email'), { exact: true }).fill(email)
  await chooseCompany(page, ORG_NAME)
  // Role deliberately left unset
  await drawer.getByRole('button', { name: t('users.create_user') }).click()

  await expect(
    drawer.getByText(t('users.user_role_ids_at_least_one_role_is_required')),
  ).toBeVisible()

  // Rejected in the browser: nothing reached the backend, so no account and no
  // welcome email.
  await expect(drawer).toBeVisible()
  expect(await findE2eUser(email), 'a rejected form must not create anything').toBeUndefined()
})
