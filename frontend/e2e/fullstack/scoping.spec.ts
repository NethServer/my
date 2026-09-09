//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Hierarchy scoping of the list views: what a persona is shown, not what it may
 * call.
 *
 * `backend/authz/` proves the API refuses to hand one company another's
 * records. This proves the other half — that the table an operator reads is
 * scoped to their own branch — which is the half a leak would actually be
 * noticed through, and the one thing a permission check cannot express: every
 * distributor holds exactly the same `read:resellers`, and the answer still has
 * to differ per distributor.
 *
 * Expectations are read off the fixture tree in `backend/authz/fixture.yml`,
 * restated below, never off the page. That file is deliberately shaped for
 * this: it has a sibling reseller under the same distributor (`d1r2`) and a
 * second branch entirely (`d2`), because isolation can only be proved against
 * an organization at the same level on the other side of the tree.
 *
 *          owner (Nethesis)
 *          ├── d1 ─────────── d1r1 ──┬── d1r1c1
 *          │     │                   └── d1r1c2
 *          │     └────────── d1r2 ───── d1r2c1
 *          └── d2 ─────────── d2r1 ───── d2r1c1
 *
 * Read-only: nothing here creates or deletes anything, so it needs no prefix
 * discipline and no teardown.
 */

import { test, expect, type Page } from '@playwright/test'
import { fixtureOrg, persona, storageStatePath } from '../fixtures/personas'
import { openAs } from '../fixtures/auth'
import { rowWithLink, rowWithText } from '../fixtures/rows'
import { t } from '../fixtures/i18n'

type Section = {
  href: string
  api: RegExp
  /** Placeholder of the section's text filter, used to defeat pagination. */
  filterKey: string
}

const RESELLERS: Section = {
  href: '/resellers',
  api: /\/api\/resellers(\?|$)/,
  filterKey: 'resellers.filter_resellers',
}

const CUSTOMERS: Section = {
  href: '/customers',
  api: /\/api\/customers(\?|$)/,
  filterKey: 'customers.filter_customers',
}

/**
 * One case per (persona, section). `visible` and `hidden` are fixture org keys
 * from the tree above.
 *
 * Only partner personas: the owner organization sees everything, so it can
 * prove nothing about scope, and a customer organization holds no hierarchy
 * permission at all — `rbac.spec.ts` covers its refusal.
 */
const CASES: { who: string; section: Section; visible: string[]; hidden: string[] }[] = [
  // A distributor reaches both of its resellers and neither of the other branch's.
  { who: 'authz-d1-admin', section: RESELLERS, visible: ['d1r1', 'd1r2'], hidden: ['d2r1'] },

  // And every customer underneath either of them — two levels down, which is
  // what makes this a subtree walk rather than a parent check.
  {
    who: 'authz-d1-admin',
    section: CUSTOMERS,
    visible: ['d1r1c1', 'd1r1c2', 'd1r2c1'],
    hidden: ['d2r1c1'],
  },

  // A reseller reaches its own customers only. `d1r2c1` is the one that matters:
  // same distributor, sibling reseller.
  {
    who: 'authz-d1r1-admin',
    section: CUSTOMERS,
    visible: ['d1r1c1', 'd1r1c2'],
    hidden: ['d1r2c1', 'd2r1c1'],
  },

  // The second branch, asserted from its own side: a scope bug that leaked
  // downwards would pass every case above and fail here.
  { who: 'authz-d2-admin', section: RESELLERS, visible: ['d2r1'], hidden: ['d1r1', 'd1r2'] },
  {
    who: 'authz-d2r1-admin',
    section: CUSTOMERS,
    visible: ['d2r1c1'],
    hidden: ['d1r1c1', 'd1r1c2', 'd1r2c1'],
  },
]

/**
 * Type a name into the section's filter and hand back the row for that
 * organization.
 *
 * The filter is what makes this pagination-proof: a distributor's customers can
 * span more than one page, and "is it listed" would otherwise mean "does it
 * happen to sort onto the first one". The row is then found by its detail link
 * rather than by text, since the filter matches substrings and the fixture's
 * names nest — see `fixtures/rows.ts`.
 */
async function filterTo(page: Page, section: Section, org: { name: string; logtoId: string }) {
  await page.getByPlaceholder(t(section.filterKey)).fill(org.name)
  return rowWithLink(page, `${section.href}/${org.logtoId}`)
}

for (const { who, section, visible, hidden } of CASES) {
  const subject = persona(who)

  test.describe(`${subject.orgName} / ${subject.userRoles.join(', ')} on ${section.href}`, () => {
    test.use({ storageState: storageStatePath(who) })

    test('lists its own branch and nothing outside it', async ({ page }) => {
      await openAs(page, section.href, section.api)

      // Its own branch first. This is also the positive control for the
      // absences below: a page that rendered nothing at all, or a filter that
      // matched nothing, would satisfy every `toHaveCount(0)` on its own.
      for (const key of visible) {
        const org = fixtureOrg(key)
        const row = await filterTo(page, section, org)

        await expect(row, `${who} should be shown ${org.name} (${key})`).toHaveCount(1, {
          timeout: 30_000,
        })
      }

      for (const key of hidden) {
        const org = fixtureOrg(key)
        const row = await filterTo(page, section, org)

        await expect(
          row,
          `${org.name} (${key}) is outside ${who}'s branch and must not be listed`,
        ).toHaveCount(0, { timeout: 30_000 })
      }
    })
  })
}

/**
 * The same question for users, where the answer is narrower: a customer
 * organization is a leaf, so it sees its own members and nobody else's — not
 * its reseller's, not a sibling customer's.
 *
 * Matched by email, which carries the fixture key (`email_template` in
 * `fixture.yml`) and so identifies one account exactly.
 */
test.describe('a customer organization on /users', () => {
  const subject = 'authz-d1r1c1-admin'

  test.use({ storageState: storageStatePath(subject) })

  test('lists its own members only', async ({ page }) => {
    await openAs(page, '/users', /\/api\/users(\?|$)/)

    const own = ['authz-d1r1c1-admin', 'authz-d1r1c1-support', 'authz-d1r1c1-backoffice']
    const others = ['authz-d1r1c2-admin', 'authz-d1r1-admin', 'authz-d1-admin']

    for (const key of own) {
      const email = persona(key).email
      await page.getByPlaceholder(t('users.filter_users')).fill(email)

      await expect(rowWithText(page, email), `${subject} should be shown ${email}`).toHaveCount(1, {
        timeout: 30_000,
      })
    }

    for (const key of others) {
      const email = persona(key).email
      await page.getByPlaceholder(t('users.filter_users')).fill(email)

      await expect(
        rowWithText(page, email),
        `${email} is outside ${subject}'s organization and must not be listed`,
      ).toHaveCount(0, { timeout: 30_000 })
    }
  })
})
