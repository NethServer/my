//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { CUSTOMERS_KEY, getCustomers } from '@/lib/customers'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'

export const useCustomers = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [CUSTOMERS_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: getCustomers,
  })
  return { ...rest, customers: state, customersAsyncStatus: asyncStatus }
})
