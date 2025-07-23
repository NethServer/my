//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getMe } from '@/lib/me'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useAuthMe = defineQuery(() => {
  const loginStore = useLoginStore()

  ////
  // const isOwner = computed(() => {
  //   return state.value.data?.username === 'owner'
  // })

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => ['authMe'],
    query: getMe,
    enabled: () => !!loginStore.jwtToken,
  })
  return {
    ...rest,
    me: state,
    meAsyncStatus: asyncStatus,
    // isOwner, ////
  }
})
