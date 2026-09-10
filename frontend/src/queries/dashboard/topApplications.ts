//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getTopApplications, TOP_APPLICATIONS_KEY } from '@/lib/dashboard/widgetData'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

export const useTopApplications = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageSize = ref(5)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [TOP_APPLICATIONS_KEY, pageSize.value],
    enabled: () => !!loginStore.jwtToken,
    query: () => getTopApplications(pageSize.value),
  })

  return {
    ...rest,
    state,
    asyncStatus,
    pageSize,
  }
})
