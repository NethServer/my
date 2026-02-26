//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getOrganizations, ORGANIZATIONS_KEY } from '@/lib/organizations/organizations'
import { useLoginStore } from '@/stores/login'
import { defineQueryOptions } from '@pinia/colada'

const loginStore = useLoginStore()

// This is not a "defineQuery" because we need an enabled condition that depends on the component that uses it (this query is typically used in a drawer that is not always open, and we don't want to fetch organizations if the drawer is not open)

export const organizationsQuery = defineQueryOptions({
  key: [ORGANIZATIONS_KEY],
  enabled: !!loginStore.jwtToken,
  query: getOrganizations,
})
