//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getOrganizations, ORGANIZATIONS_KEY } from '@/lib/organizations/organizations'
import { defineQueryOptions } from '@pinia/colada'

// This is not a "defineQuery" because some consumers override the enabled
// condition based on local UI state, such as whether a drawer is open.

export const organizationsQuery = defineQueryOptions({
  key: [ORGANIZATIONS_KEY],
  query: getOrganizations,
})
