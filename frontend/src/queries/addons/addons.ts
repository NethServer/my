//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { ADDONS_KEY, getAddons } from '@/lib/addons/addons'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

// The catalog endpoint returns the whole list in one unpaginated, unfiltered
// response, so this query takes no parameters: searching, filtering and paging
// all happen client-side in the catalog panel.
export const useAddons = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [ADDONS_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getAddons(),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
