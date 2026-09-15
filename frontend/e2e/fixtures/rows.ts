//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Finding the table row for one particular thing.
 *
 * Harder than it looks, and worth doing properly. `hasText` matches a row's
 * whole text content, in which the cells are concatenated with no separator —
 * so "Company name" and the value that follows it run together as
 * `Company nameauthz-d1r1c1`, and a word-boundary pattern cannot see the seam.
 * Meanwhile the authz fixture's names nest on purpose (`authz-d1r1` is a prefix
 * of `authz-d1r1c1`, see `backend/authz/README.md`), and an organization row
 * also carries the name of the organization that *created* it — so a plain
 * substring finds `authz-d1r1` in a row that is really about its own customer.
 *
 * Both problems disappear by matching on something with structure instead.
 */

import type { Locator, Page } from '@playwright/test'

/**
 * The row whose cells contain a link to `href` — for organizations, the detail
 * link the name is rendered as (`OrganizationLink.vue`), which carries the
 * Logto id and so identifies exactly one row whatever it is named.
 */
export function rowWithLink(page: Page, href: string): Locator {
  return page.getByRole('row').filter({ has: page.locator(`a[href="${href}"]`) })
}

/**
 * The row containing `text` anywhere in it.
 *
 * Only for values that cannot appear inside a longer one or in another row's
 * cells — an email address, which ends at its domain, or the suite's own `e2e-`
 * marker where the point is to count the rows that carry it. Never for a
 * fixture organization name; use `rowWithLink`.
 */
export function rowWithText(page: Page, text: string): Locator {
  return page.getByRole('row').filter({ hasText: text })
}
