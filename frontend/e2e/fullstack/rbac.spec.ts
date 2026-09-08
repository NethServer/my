//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * The permission-driven UI, asserted for every (organization role x technical
 * role) persona.
 *
 * `backend/authz/` already proves the API refuses the wrong caller. This proves
 * the other half: that the interface a persona is shown matches the permissions
 * the backend signed for them — no entry offered that their token cannot use,
 * and none withheld that it can.
 *
 * The same rule the authorization suite works under applies here (see
 * `backend/authz/README.md`): NEVER derive an expectation from the component
 * under test. `NAV` below states what each entry *should* require, taken from
 * the permission vocabulary in `lib/permissions.ts` and the documented intent of
 * each gate. If it disagrees with `SideMenu.vue`, that disagreement is the
 * finding — copying the component's own conditions in would make this prove
 * nothing.
 *
 * The permission set itself is not restated: it is captured from the token
 * exchange, so these are the grants the server really issued.
 */

import { test, expect } from '@playwright/test'
import { matrixPersonas, storageStatePath } from '../fixtures/personas'
import { effectivePermissions, hasOwnerLevelAuthority, openAs } from '../fixtures/auth'

/** Sentinel for gates that answer to owner-level authority, not a permission. */
const OWNER_LEVEL = Symbol('owner-level authority')

/**
 * Every navigation entry the shell can offer, and what it should require.
 * `null` means unconditional. Keyed by href because `router-link` renders one,
 * which keeps these selectors independent of the interface language.
 */
const NAV: { href: string; requires: string | typeof OWNER_LEVEL | null }[] = [
  { href: '/dashboard', requires: null },
  { href: '/alerts', requires: 'read:systems' },
  { href: '/systems', requires: 'read:systems' },
  { href: '/applications', requires: 'read:applications' },
  // Curating the add-on catalog is a licensing back-office duty, not something
  // every company that can see its own add-ons takes part in.
  { href: '/addons', requires: OWNER_LEVEL },
  { href: '/distributors', requires: 'read:distributors' },
  { href: '/resellers', requires: 'read:resellers' },
  { href: '/customers', requires: 'read:customers' },
  { href: '/users', requires: 'read:users' },
]

for (const who of matrixPersonas) {
  test.describe(`${who.orgRole} / ${who.userRoles.join(', ')} (${who.key})`, () => {
    test.use({ storageState: storageStatePath(who.key) })

    test('is offered exactly the navigation its permissions allow', async ({ page }) => {
      const user = await openAs(page, '/dashboard')
      const granted = effectivePermissions(user)
      const ownerLevel = hasOwnerLevelAuthority(user)

      // Guard against a vacuous pass: a persona whose exchange returned no
      // permissions at all would trivially satisfy every "hidden" assertion.
      expect(
        granted.length,
        `${who.key} was signed in with no permissions at all — the fixture is stale, ` +
          'reprovision with `./apitool authz provision`',
      ).toBeGreaterThan(0)

      const nav = page.getByRole('navigation')
      await expect(nav).toBeVisible()

      for (const entry of NAV) {
        const shouldSee =
          entry.requires === null
            ? true
            : entry.requires === OWNER_LEVEL
              ? ownerLevel
              : granted.includes(entry.requires as string)

        const link = nav.locator(`a[href="${entry.href}"]`)
        const because =
          entry.requires === OWNER_LEVEL
            ? `owner-level authority (org_role=${user.org_role}, roles=[${user.user_roles.join(', ')}])`
            : (entry.requires ?? 'nothing')

        if (shouldSee) {
          await expect(
            link,
            `${entry.href} needs ${String(because)}, which this persona has`,
          ).toBeVisible()
        } else {
          await expect(
            link,
            `${entry.href} needs ${String(because)}, which this persona lacks`,
          ).toHaveCount(0)
        }
      }
    })

    test('refuses a deep link the persona has no permission for', async ({ page }) => {
      const user = await openAs(page, '/dashboard')
      const granted = effectivePermissions(user)

      // Pick a section this persona genuinely cannot read. Nothing to prove for
      // the owner, who can read everything.
      const forbidden = NAV.find(
        (e) => typeof e.requires === 'string' && !granted.includes(e.requires),
      )

      test.skip(!forbidden, 'this persona can read every section')

      // The router guard only checks authentication, so the view renders and
      // its first request is what gets refused; the 403 interceptor in
      // lib/axios.ts is what turns that into a redirect.
      await page.goto(forbidden!.href)
      await expect(page).toHaveURL(/\/forbidden$/, { timeout: 30_000 })
    })
  })
}
