//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getDashboardCatalog, DASHBOARD_CATALOG_KEY } from '@/lib/dashboard/generate'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

// Drives the topic picker in the customization wizard: the backend returns only
// the widgets this user may see, so a topic that would render nothing is never
// offered.
export const useDashboardCatalog = defineQuery(() => {
  const loginStore = useLoginStore()
  // Only the wizard needs the catalog, and the wizard is mounted (hidden) on
  // every dashboard load. Without this the endpoint would be called on every
  // visit to pay for a drawer most visits never open.
  const isEnabled = ref(false)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [DASHBOARD_CATALOG_KEY],
    enabled: () => isEnabled.value && !!loginStore.jwtToken,
    query: getDashboardCatalog,
  })

  return {
    ...rest,
    state,
    asyncStatus,
    isEnabled,
  }
})
