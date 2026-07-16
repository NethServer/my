//
// Copyright (C) 2026 Nethesis S.r.l.
// SPDX-License-Identifier: GPL-3.0-or-later
//

import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'
import { useLoginStore } from '@/stores/login'
import {
  ENTITLEMENT_CATALOG_KEY,
  ENTITLEMENTS_REFETCH_INTERVAL_SECONDS,
  SYSTEM_ENTITLEMENTS_KEY,
  getAvailableEntitlements,
  getEntitlementCatalog,
  getSystemEntitlements,
} from '@/lib/entitlements/entitlements'

export const useSystemEntitlements = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()
  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [SYSTEM_ENTITLEMENTS_KEY, route.params.systemId as string],
    enabled: () => !!loginStore.jwtToken && !!route.params.systemId,
    query: () => getSystemEntitlements(route.params.systemId as string),
    // Shop webhooks (pending/activate) land asynchronously: keep polling
    // like the alerts tables do.
    staleTime: ENTITLEMENTS_REFETCH_INTERVAL_SECONDS * 1000,
    autoRefetch: true,
  })
  return { ...rest, state, asyncStatus }
})

export const useAvailableEntitlements = defineQuery(() => {
  const loginStore = useLoginStore()
  const { state, asyncStatus, ...rest } = useQuery({
    key: () => ['available_entitlements'],
    enabled: () => !!loginStore.jwtToken,
    query: () => getAvailableEntitlements(),
  })
  return { ...rest, state, asyncStatus }
})

export const useEntitlementCatalog = defineQuery(() => {
  const loginStore = useLoginStore()
  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [ENTITLEMENT_CATALOG_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getEntitlementCatalog(),
  })
  return { ...rest, state, asyncStatus }
})
