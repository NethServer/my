//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getUserRoles, USER_ROLES_KEY } from '@/lib/userRoles'
import { useLoginStore } from '@/stores/login'
import { defineQueryOptions } from '@pinia/colada'

const loginStore = useLoginStore()

// This is not a "defineQuery" because we need an enabled condition that depends on the component that uses it (this query is typically used in a drawer that is not always open, and we don't want to fetch user roles if the drawer is not open)

export const userRolesQuery = defineQueryOptions({
  key: [USER_ROLES_KEY],
  enabled: !!loginStore.jwtToken,
  query: getUserRoles,
})
