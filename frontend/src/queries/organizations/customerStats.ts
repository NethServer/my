//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { canReadCustomers } from '@/lib/permissions'
import { getCustomerStats } from '@/lib/organizations/customerDetail'
import { CUSTOMERS_KEY } from '@/lib/organizations/customers'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'

export const useCustomerStats = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [CUSTOMERS_KEY, 'stats', route.params.companyId],
    enabled: () => !!loginStore.jwtToken && canReadCustomers() && !!route.params.companyId,
    query: () => getCustomerStats(route.params.companyId as string),
  })

  return {
    ...rest,
    state,
    asyncStatus,
  }
})
