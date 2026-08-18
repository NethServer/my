//
// Copyright (C) 2026 Nethesis S.r.l.
// SPDX-License-Identifier: GPL-3.0-or-later
//

import { defineQuery, useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useLoginStore } from '@/stores/login'
import { MIN_SEARCH_LENGTH } from '@/lib/common'
import {
  ENTITLEMENT_CATALOG_KEY,
  ENTITLEMENT_REPORT_KEY,
  ENTITLEMENTS_REFETCH_INTERVAL_SECONDS,
  SYSTEM_ENTITLEMENTS_KEY,
  getAvailableEntitlements,
  getEntitlementCatalog,
  getEntitlementReport,
  getEntitlementReportOrganizations,
  getEntitlementReportTiers,
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

export const useEntitlementReport = defineQuery(() => {
  const loginStore = useLoginStore()
  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [ENTITLEMENT_REPORT_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getEntitlementReport(),
  })
  return { ...rest, state, asyncStatus }
})

// Server-paginated + searchable report slice, same shape as the systems
// query composable: debounced text filter, page reset on filter/size change.
function defineReportSliceQuery<T>(
  keyName: string,
  fetcher: (page: number, pageSize: number, search: string) => Promise<T>,
) {
  return defineQuery(() => {
    const loginStore = useLoginStore()
    const pageNum = ref(1)
    const pageSize = ref(10)
    const textFilter = ref('')
    const debouncedTextFilter = ref('')

    const { state, asyncStatus, ...rest } = useQuery({
      key: () => [
        keyName,
        {
          pageNum: pageNum.value,
          pageSize: pageSize.value,
          textFilter: debouncedTextFilter.value,
        },
      ],
      enabled: () => !!loginStore.jwtToken,
      query: () => fetcher(pageNum.value, pageSize.value, debouncedTextFilter.value),
      placeholderData: (previous) => previous,
    })

    watch(
      () => textFilter.value,
      useDebounceFn(() => {
        // debounce and ignore if text filter is too short
        if (textFilter.value.length === 0 || textFilter.value.length >= MIN_SEARCH_LENGTH) {
          debouncedTextFilter.value = textFilter.value
          pageNum.value = 1
        }
      }, 500),
    )

    watch(
      () => pageSize.value,
      () => {
        pageNum.value = 1
      },
    )

    return { ...rest, state, asyncStatus, pageNum, pageSize, textFilter }
  })
}

export const useEntitlementReportOrganizations = defineReportSliceQuery(
  'entitlement_report_organizations',
  getEntitlementReportOrganizations,
)

export const useEntitlementReportTiers = defineReportSliceQuery(
  'entitlement_report_tiers',
  getEntitlementReportTiers,
)
