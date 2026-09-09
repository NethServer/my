//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

/**
 * Helpers for specs that act as a specific persona.
 *
 * `openAs` boots the application with that persona's saved session and hands
 * back the user record the backend signed — the same object the application
 * itself drives its permission gates from, captured off the wire rather than
 * re-derived, so a spec asserts against what the server actually granted.
 */

import type { Page } from '@playwright/test'

/** Shape of `data.user` in the `POST /api/auth/exchange` response. */
export type SignedInUser = {
  email: string
  name: string
  org_role: string
  org_permissions: string[]
  user_permissions: string[]
  user_roles: string[]
  organization_name: string
}

/** Effective permission set, exactly as `stores/login.ts` computes it. */
export function effectivePermissions(user: SignedInUser): string[] {
  return [...(user.org_permissions ?? []), ...(user.user_permissions ?? [])]
}

/**
 * "Owner-level authority" as `lib/permissions.ts` defines it: the Owner
 * organization, or a Super Admin user. Not a permission — a threshold above the
 * `manage:*` scopes every distributor already holds.
 */
export function hasOwnerLevelAuthority(user: SignedInUser): boolean {
  return user.org_role === 'Owner' || (user.user_roles ?? []).includes('Super Admin')
}

/**
 * A promise for the next response whose URL matches `match`.
 *
 * Always arm this *before* the navigation, click or reload that triggers the
 * request. `page.waitForResponse` only sees traffic that arrives after it
 * starts listening, so arming it afterwards waits for a second request that
 * never comes — and it only fails when the response is quick, which reads as
 * flakiness rather than as the ordering bug it is.
 */
export function apiResponse(page: Page, match: RegExp) {
  return page.waitForResponse((r) => match.test(r.url()), { timeout: 60_000 })
}

/**
 * Navigate to `path` as the persona whose session the test is using, and return
 * the signed-in user captured from the token exchange that boots the session.
 *
 * Pass `waitFor` to also wait for the request the page fires on mount, so the
 * spec acts on a rendered list instead of a skeleton.
 */
export async function openAs(
  page: Page,
  path = '/dashboard',
  waitFor?: RegExp,
): Promise<SignedInUser> {
  const exchange = page.waitForResponse(
    (r) => r.url().includes('/auth/exchange') && r.status() === 200,
    { timeout: 60_000 },
  )
  const listed = waitFor ? apiResponse(page, waitFor) : undefined

  await page.goto(path)

  const body = (await (await exchange).json()) as { data: { user: SignedInUser } }
  await listed
  return body.data.user
}
