//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canReadApplications } from '@/lib/permissions'
import { getApplicationDetail } from '@/lib/applications/applicationDetail'
import { APPLICATIONS_KEY } from '@/lib/applications/applications'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useApplicationDetail = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [APPLICATIONS_KEY, route.params.applicationId],
    enabled: () => !!loginStore.jwtToken && canReadApplications() && !!route.params.applicationId,
    query: () => getApplicationDetail(route.params.applicationId as string),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
