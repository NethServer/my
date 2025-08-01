//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getUsers, USERS_KEY } from '@/lib/users'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useUsers = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [USERS_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: getUsers,
  })
  return { ...rest, users: state, usersAsyncStatus: asyncStatus }
})
