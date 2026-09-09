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
import { fixtureOrg, owner, storageStatePath } from '../fixtures/personas'
import { openAs } from '../fixtures/auth'
import { t } from '../fixtures/i18n'
import {
  MAIL_TAG,
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
const ORG_NAME = fixtureOrg('d1r1c1').name

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
  await openAs(page, '/users', /\/api\/users(\?|$)/)
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
 *
 * Waiting for the target row alone would not prove the filter ran: on a page
 * that already shows it, the assertion passes before a keystroke has any
 * effect. Wait for the narrowed state — exactly one row carrying the suite's
 * address tag, and it is the right one.
 */
async function filterTo(page: Page, email: string): Promise<Locator> {
  await page.getByPlaceholder(t('users.filter_users')).fill(email)

  const rows = page.getByRole('row').filter({ hasText: MAIL_TAG })
  await expect(rows).toHaveCount(1, { timeout: 30_000 })
  await expect(rows.first()).toContainText(email)
  return rows.first()
}

test.describe('user lifecycle', () => {
  // Serial for the reason in the file docstring: one account covers create,
  // edit and archive, so each step depends on the one before. Scoped to this
  // describe rather than the file, so a failure here does not skip the
  // independent validation test below.
  //
  // `email` is computed once per worker process, so a retry of the group reuses
  // the address — which is fine only because `beforeAll` re-runs and sweeps it
  // first. Load-bearing, hence spelled out.
  test.describe.configure({ mode: 'serial' })

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
    const row = await filterTo(page, email)
    await expect(row).toContainText(t('user_roles.reader'))

    // The row is one thing; what the backend stored is another.
    const stored = await findE2eUser(email)
    expect(stored, `${email} should exist in the backend after creating it`).toBeDefined()
    expect(stored?.roles?.map((r) => r.id)).toEqual([await roleIdByName('Reader')])
    expect(stored?.organization?.name, 'the user should land in the chosen company').toBe(ORG_NAME)
  })

  test('changes the assigned role and keeps the change', async ({ page }) => {
    await openUsers(page)

    const row = await filterTo(page, email)
    await row.getByRole('button', { name: t('common.edit') }).click()

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

    const row = await filterTo(page, email)
    await row.getByRole('button', { name: /menu/i }).click()
    await page.getByRole('menuitem', { name: t('common.archive') }).click()

    // The confirmation is type-to-confirm, and that is load-bearing:
    // `DeleteObjectModal.vue` only emits primary-click when the typed text
    // matches the name exactly, so clicking Archive without it shows a
    // validation error and archives nothing.
    //
    // Asserted on the heading rather than on the dialog itself: the element
    // carrying role="dialog" is a zero-size wrapper around fixed-position
    // children, which Playwright reports as hidden however open the modal is.
    const confirm = page.getByRole('dialog')
    const title = confirm.getByRole('heading', { name: t('users.archive_user') })

    await expect(title).toBeVisible()
    await confirm.getByRole('textbox').fill(name)
    await confirm.getByRole('button', { name: t('common.archive') }).click()

    // Wait for the modal to go before looking at the table, and not only for
    // tidiness: while it is open the rest of the page is aria-hidden, so
    // `getByRole('row')` matches nothing and the assertion below would pass
    // against an archive that never happened.
    await expect(title).toBeHidden({ timeout: 30_000 })

    // Archived users drop out of the listing, which asks for enabled and
    // suspended only.
    await expect(rowOf(page, email)).toHaveCount(0, { timeout: 30_000 })

    // The row leaving the page is not the same as the account being archived.
    // Ask the backend for the archived ones and it is there.
    const archived = await listE2eUsers(['deleted'])
    expect(
      archived.map((u) => u.email),
      `${email} should be listed once the archived are asked for`,
    ).toContain(email)
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

  // The drawer stays open, so the form was rejected in the browser and nothing
  // was submitted — no account, no welcome email. The backend lookup that
  // follows cannot fail while that holds: it guards against a form that starts
  // submitting anyway, and is not evidence that the server refuses anything.
  await expect(drawer).toBeVisible()
  expect(await findE2eUser(email), 'a rejected form must not create anything').toBeUndefined()
})
