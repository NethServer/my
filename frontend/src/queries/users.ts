//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getUsers } from '@/lib/users'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useUsers = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => ['users'],
    enabled: () => !!loginStore.jwtToken,
    query: getUsers,
  })
  return { ...rest, users: state, usersAsyncStatus: asyncStatus }
})
