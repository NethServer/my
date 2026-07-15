//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  getAlerts,
  ALERTS_ALERTS_KEY,
  ALERTS_TABLE_ID,
  type AlertSortBy,
  type AlertStatusEnum,
  ALERTS_REFETCH_INTERVAL_SECONDS,
} from '@/lib/alerts'
import { syncWithBackend } from '@/lib/alertPendingStates'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { NeDropdownFilterV2Option } from '@nethesis/vue-components'

export const useAlerts = defineQuery(() => {
  const loginStore = useLoginStore()
  const { t } = useI18n()
  const organizationIds = ref<NeDropdownFilterV2Option[]>([])
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const sortBy = ref<AlertSortBy>('starts_at')
  const sortDirection = ref<'asc' | 'desc'>('desc')
  // Default: show only unmuted (active) alerts.
  const defaultStatusFilters = (): NeDropdownFilterV2Option[] => [
    { id: 'active', label: t('alerts.unmuted') },
  ]
  const isDefaultStatusFilter = () =>
    statusFilters.value.length === 1 && statusFilters.value[0].id === 'active'
  const statusFilters = ref<NeDropdownFilterV2Option[]>(defaultStatusFilters())
  const severityFilters = ref<NeDropdownFilterV2Option[]>([])
  const systemKeyFilters = ref<NeDropdownFilterV2Option[]>([])
  const alertnameFilters = ref<NeDropdownFilterV2Option[]>([])
  const assigneeFilters = ref<NeDropdownFilterV2Option[]>([])
  const shouldAutoRefetch = () => document.visibilityState === 'visible'

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ALERTS_ALERTS_KEY,
      organizationIds.value.map((o) => o.id).join(','),
      pageNum.value,
      pageSize.value,
      sortBy.value,
      sortDirection.value,
      statusFilters.value.map((o) => o.id).join(','),
      severityFilters.value.map((o) => o.id).join(','),
      systemKeyFilters.value.map((o) => o.id).join(','),
      alertnameFilters.value.map((o) => o.id).join(','),
      assigneeFilters.value.map((o) => o.id).join(','),
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getAlerts(
        organizationIds.value.length > 0 ? organizationIds.value.map((o) => o.id) : undefined,
        pageNum.value,
        pageSize.value,
        sortBy.value,
        sortDirection.value,
        statusFilters.value.length > 0
          ? (statusFilters.value.map((o) => o.id) as AlertStatusEnum[])
          : undefined,
        severityFilters.value.length > 0 ? severityFilters.value.map((o) => o.id) : undefined,
        systemKeyFilters.value.length > 0 ? systemKeyFilters.value.map((o) => o.id) : undefined,
        alertnameFilters.value.length > 0 ? alertnameFilters.value.map((o) => o.id) : undefined,
        assigneeFilters.value.length > 0
          ? assigneeFilters.value.map((o) => String(o.id))
          : undefined,
      ),
    staleTime: ALERTS_REFETCH_INTERVAL_SECONDS * 1000,
    autoRefetch: shouldAutoRefetch,
  })

  const clearFilters = () => {
    organizationIds.value = []
    severityFilters.value = []
    systemKeyFilters.value = []
    alertnameFilters.value = []
    assigneeFilters.value = []
    resetStatusFilter()
    pageNum.value = 1
  }

  const resetStatusFilter = () => {
    statusFilters.value = defaultStatusFilters()
  }

  const areDefaultFiltersApplied = () => {
    return (
      !organizationIds.value.length &&
      isDefaultStatusFilter() &&
      !severityFilters.value.length &&
      !systemKeyFilters.value.length &&
      !alertnameFilters.value.length &&
      !assigneeFilters.value.length
    )
  }

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(ALERTS_TABLE_ID)
      }
    },
    { immediate: true },
  )

  // reset to first page when page size changes
  watch(
    () => pageSize.value,
    () => {
      pageNum.value = 1
    },
  )

  // reset to first page when status filter changes
  watch(
    () => statusFilters.value,
    () => {
      pageNum.value = 1
    },
    { deep: true },
  )

  // When the backend returns fresh data, clean up pending states that are now confirmed
  watch(
    () => state.value.data?.alerts,
    (alerts) => {
      if (alerts) syncWithBackend(alerts)
    },
  )

  return {
    ...rest,
    state,
    asyncStatus,
    organizationIds,
    pageNum,
    pageSize,
    sortBy,
    sortDirection,
    statusFilters,
    severityFilters,
    systemKeyFilters,
    alertnameFilters,
    assigneeFilters,
    clearFilters,
    resetStatusFilter,
    areDefaultFiltersApplied,
  }
})
