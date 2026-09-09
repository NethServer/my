//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * The controls inside a page, asserted for every persona — the half of "the UI
 * hides the button" that navigation coverage does not reach.
 *
 * `rbac.spec.ts` proves a persona is offered the right sections. This proves
 * that once inside one, it is offered the right actions: a Reader may open
 * /systems and must not be invited to create one; a distributor Admin manages
 * resellers and must not be offered Promote, which moves an organization out of
 * its own reach.
 *
 * The same rule as the RBAC spec (see `backend/authz/README.md`): NEVER derive
 * an expectation from the component under test. `SURFACES` states what each
 * control *should* require, taken from the permission vocabulary in
 * `lib/permissions.ts` and the documented intent of each gate. A disagreement
 * with the table code is the finding.
 *
 * Every surface names one control that must be there for anyone who can open
 * it. Without that, a page which failed to render — or a kebab which failed to
 * open — would satisfy every absence assertion below it.
 *
 * Read-only: the controls are located, never clicked, so nothing is created and
 * there is nothing to tear down. Two gates are deliberately left out and
 * tracked as later work, both because they need a fixture this spec would have
 * to build: impersonation (`impersonate:users`, which also depends on the
 * target's consent) and the add-on catalog, including the compound
 * `canBuyAddons` — `manage:entitlements` AND NOT owner-level, the one gate in
 * the codebase where holding more authority has to hide a control.
 */

import { test, expect, type Locator, type Page } from '@playwright/test'
import { fixtureOrg, matrixPersonas, storageStatePath, type Persona } from '../fixtures/personas'
import { effectivePermissions, hasOwnerLevelAuthority, openAs } from '../fixtures/auth'
import { rowWithLink } from '../fixtures/rows'
import { t } from '../fixtures/i18n'

/** Sentinel for gates that answer to owner-level authority, not a permission. */
const OWNER_LEVEL = Symbol('owner-level authority')

type Requirement = string | typeof OWNER_LEVEL

type Control = {
  /** What it is, for the failure message. */
  what: string
  requires: Requirement
  locate: (page: Page) => Locator
}

type Surface = {
  href: string
  /**
   * The list request the page fires on mount, waited for so assertions run
   * against a settled page. Omitted where the controls under test come from the
   * permission set alone and no request is involved.
   */
  api?: RegExp
  /**
   * Everything a persona must hold for this surface to be reachable *with its
   * positive control present*. Usually just the read permission; the kebab
   * surface below also needs whatever puts an item in the menu, since a persona
   * with no items gets no menu to open and nothing to assert about.
   *
   * A persona missing any of these skips the surface. What it is then denied is
   * covered by whichever surface it can reach: a Reader gets no kebab on
   * /resellers, and the create-reseller button on the same page is where its
   * lack of `manage:resellers` is asserted.
   */
  open: Requirement[]
  /** Present for every persona that can open the surface. The positive control. */
  always: { what: string; locate: (page: Page) => Locator }
  controls: Control[]
  /** Run before locating anything, e.g. to open a row's kebab menu. */
  prepare?: (page: Page, subject: Persona) => Promise<void>
}

const button = (key: string) => (page: Page) => page.getByRole('button', { name: t(key) })
const menuitem = (key: string) => (page: Page) => page.getByRole('menuitem', { name: t(key) })
const heading = (key: string) => (page: Page) => page.getByRole('heading', { name: t(key) })

/**
 * One tab of a `NeTabs`, by its label inside the tab bar.
 *
 * Scoped to the tab bar's own navigation landmark — the component labels it with
 * `srTabsLabel` — because the desktop tab is an `<a>` with no href and so has no
 * role of its own to query.
 */
const tab = (key: string) => (page: Page) =>
  page.getByRole('navigation', { name: t('ne_tabs.tabs') }).getByText(t(key), { exact: true })

/**
 * A reseller this persona can actually see, so the kebab menu has a row to open.
 * Only the Owner organization and a distributor hold `read:resellers`, and a
 * distributor sees its own branch only — hence the second branch's own reseller
 * for a `d2` persona.
 */
function visibleReseller(subject: Persona): string {
  return subject.orgName === fixtureOrg('d2').name ? 'd2r1' : 'd1r1'
}

const SURFACES: Surface[] = [
  {
    href: '/systems',
    api: /\/api\/systems(\?|$)/,
    open: ['read:systems'],
    always: { what: 'the page heading', locate: heading('systems.title') },
    controls: [
      {
        what: 'the create-system button',
        requires: 'manage:systems',
        locate: button('systems.create_system'),
      },
    ],
  },
  {
    href: '/users',
    api: /\/api\/users(\?|$)/,
    open: ['read:users'],
    always: { what: 'the page heading', locate: heading('users.title') },
    controls: [
      {
        what: 'the create-user button',
        requires: 'manage:users',
        locate: button('users.create_user'),
      },
    ],
  },
  {
    href: '/distributors',
    api: /\/api\/distributors(\?|$)/,
    open: ['read:distributors'],
    always: { what: 'the page heading', locate: heading('distributors.title') },
    controls: [
      {
        what: 'the create-distributor button',
        requires: 'manage:distributors',
        locate: button('distributors.create_distributor'),
      },
    ],
  },
  {
    href: '/customers',
    api: /\/api\/customers(\?|$)/,
    open: ['read:customers'],
    always: { what: 'the page heading', locate: heading('customers.title') },
    controls: [
      {
        what: 'the create-customer button',
        requires: 'manage:customers',
        locate: button('customers.create_customer'),
      },
    ],
  },
  {
    href: '/resellers',
    api: /\/api\/resellers(\?|$)/,
    open: ['read:resellers'],
    always: { what: 'the page heading', locate: heading('resellers.title') },
    controls: [
      {
        what: 'the create-reseller button',
        requires: 'manage:resellers',
        locate: button('resellers.create_reseller'),
      },
    ],
  },
  {
    // The same page again, with a row's kebab open. Separate rather than folded
    // in, so the preparation belongs to the assertions that need it.
    href: '/resellers',
    api: /\/api\/resellers(\?|$)/,
    // Also `manage:resellers`, which is what puts Edit in the menu — and is not
    // implied by `read:resellers`: the backend subtracts the organization role's
    // `manage:*` from anyone holding the Reader role, a deviation from
    // "effective = org ∪ user" recorded in `backend/authz/model.yml`
    // (`filterManagePermissionsForReader`). A distributor Reader therefore has
    // no kebab at all, and no menu to open.
    open: ['read:resellers', 'manage:resellers'],
    prepare: async (page, subject) => {
      const org = fixtureOrg(visibleReseller(subject))

      await page.getByPlaceholder(t('resellers.filter_resellers')).fill(org.name)
      const row = rowWithLink(page, `/resellers/${org.logtoId}`)
      await expect(row).toHaveCount(1, { timeout: 30_000 })
      await row.getByRole('button', { name: /menu/i }).click()
    },
    always: { what: 'the Edit item', locate: menuitem('common.edit') },
    controls: [
      {
        // Moving an organization between hierarchy levels takes it out of the
        // scope of the company that manages it, so it answers to owner-level
        // authority rather than to the manage:resellers every distributor holds.
        what: 'the Promote item',
        requires: OWNER_LEVEL,
        locate: menuitem('common.promote'),
      },
      {
        what: 'the Destroy item',
        requires: 'destroy:resellers',
        locate: menuitem('common.destroy'),
      },
    ],
  },
  {
    href: '/alerts',
    // No api: the tabs come from `tabsConfig` in AlertsView, computed from the
    // permission set, so there is no request to wait for and no need to guess
    // which endpoint the active-alerts panel calls.
    //
    // The entry itself answers to read:systems — active alerts are system data.
    // The alerting *configuration* is what read:alerts gates, and it lives on
    // the second tab, which is the gate under test here.
    open: ['read:systems'],
    always: { what: 'the active-alerts tab', locate: tab('alerts.active_alerts_tab') },
    controls: [
      {
        what: 'the notifications tab',
        requires: 'read:alerts',
        locate: tab('alerts.notifications_tab'),
      },
    ],
  },
]

function holds(granted: string[], ownerLevel: boolean, requires: Requirement): boolean {
  return requires === OWNER_LEVEL ? ownerLevel : granted.includes(requires)
}

function describeRequirement(requires: Requirement, user: { org_role: string }): string {
  return requires === OWNER_LEVEL ? `owner-level authority (org_role=${user.org_role})` : requires
}

for (const who of matrixPersonas) {
  test.describe(`${who.orgRole} / ${who.userRoles.join(', ')} (${who.key})`, () => {
    test.use({ storageState: storageStatePath(who.key) })

    test('is offered exactly the actions its permissions allow', async ({ page }) => {
      const user = await openAs(page, '/dashboard')
      const granted = effectivePermissions(user)
      const ownerLevel = hasOwnerLevelAuthority(user)

      // Same guard as rbac.spec.ts: a persona whose exchange returned nothing
      // would satisfy every absence assertion below for free.
      expect(
        granted.length,
        `${who.key} was signed in with no permissions at all — the fixture is stale, ` +
          'reprovision with `./apitool authz provision`',
      ).toBeGreaterThan(0)

      const reachable = SURFACES.filter((s) =>
        s.open.every((need) => holds(granted, ownerLevel, need)),
      )

      expect(
        reachable.length,
        `${who.key} can open none of the surfaces this spec knows about`,
      ).toBeGreaterThan(0)

      let current: string | undefined

      for (const surface of reachable) {
        // Two surfaces share /resellers — the page, and the page with a row's
        // kebab open. Reloading between them would cost a boot and a token
        // exchange to arrive back where we already are.
        if (surface.href !== current) {
          await openAs(page, surface.href, surface.api)
          current = surface.href
        }
        await surface.prepare?.(page, who)

        // The positive control; `open` is what guarantees it is there.
        await expect(
          surface.always.locate(page),
          `${surface.href} should show ${surface.always.what} to anyone who can open it`,
        ).toBeVisible({ timeout: 30_000 })

        for (const control of surface.controls) {
          const because = describeRequirement(control.requires, user)
          const locator = control.locate(page)

          if (holds(granted, ownerLevel, control.requires)) {
            await expect(
              locator,
              `${surface.href}: ${control.what} needs ${String(because)}, which this persona has`,
            ).toBeVisible({ timeout: 30_000 })
          } else {
            await expect(
              locator,
              `${surface.href}: ${control.what} needs ${String(because)}, which this persona lacks`,
            ).toHaveCount(0, { timeout: 30_000 })
          }
        }
      }
    })
  })
}
