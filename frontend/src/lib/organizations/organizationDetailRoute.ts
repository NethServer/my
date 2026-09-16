//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { RouteLocationNamedRaw } from 'vue-router'

const DETAIL_ROUTE_BY_TYPE: Record<string, string> = {
  distributor: 'distributor_detail',
  reseller: 'reseller_detail',
  customer: 'customer_detail',
}

/**
 * Route to an organization's detail page, keyed on its Logto id and level.
 *
 * Returns null when the organization has no detail page to link to: the Owner
 * organization, an organization not synced with Logto yet (no id), or one whose
 * level could not be resolved because it was deleted.
 */
export function organizationDetailRoute(
  logtoId: string | undefined,
  type: string | undefined,
): RouteLocationNamedRaw | null {
  if (!logtoId || !type) {
    return null
  }

  const name = DETAIL_ROUTE_BY_TYPE[type.toLowerCase()]
  if (!name) {
    return null
  }

  return { name, params: { companyId: logtoId } }
}
