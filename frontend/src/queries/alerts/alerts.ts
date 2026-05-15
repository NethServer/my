//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getAlerts, ALERTS_ALERTS_KEY } from '@/lib/alerts'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { ref } from 'vue'

export const useAlerts = defineQuery(() => {
  const loginStore = useLoginStore()
  const organizationIds = ref<string[]>([])
  const pageNum = ref(1)
  const pageSize = ref(50)
  const sortBy = ref<'starts_at' | 'severity' | 'alertname' | 'status'>('starts_at')
  const sortDirection = ref<'asc' | 'desc'>('desc')
  const statusFilters = ref<string[]>([])
  const severityFilters = ref<string[]>([])
  const systemKeyFilters = ref<string[]>([])
  const alertnameFilters = ref<string[]>([])

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ALERTS_ALERTS_KEY,
      organizationIds.value.join(','),
      pageNum.value,
      pageSize.value,
      sortBy.value,
      sortDirection.value,
      statusFilters.value.join(','),
      severityFilters.value.join(','),
      systemKeyFilters.value.join(','),
      alertnameFilters.value.join(','),
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getAlerts(
        organizationIds.value.length > 0 ? organizationIds.value : undefined,
        pageNum.value,
        pageSize.value,
        sortBy.value,
        sortDirection.value,
        statusFilters.value.length > 0 ? statusFilters.value : undefined,
        severityFilters.value.length > 0 ? severityFilters.value : undefined,
        systemKeyFilters.value.length > 0 ? systemKeyFilters.value : undefined,
        alertnameFilters.value.length > 0 ? alertnameFilters.value : undefined,
      ),
  })

  const resetFilters = () => {
    statusFilters.value = []
    severityFilters.value = []
    systemKeyFilters.value = []
    alertnameFilters.value = []
    pageNum.value = 1
  }

  const areDefaultFiltersApplied = () => {
    return (
      !statusFilters.value.length &&
      !severityFilters.value.length &&
      !systemKeyFilters.value.length &&
      !alertnameFilters.value.length
    )
  }

  const toggleStatusFilter = (status: string) => {
    const idx = statusFilters.value.indexOf(status)
    if (idx >= 0) {
      statusFilters.value.splice(idx, 1)
    } else {
      statusFilters.value.push(status)
    }
    pageNum.value = 1
  }

  const toggleSeverityFilter = (severity: string) => {
    const idx = severityFilters.value.indexOf(severity)
    if (idx >= 0) {
      severityFilters.value.splice(idx, 1)
    } else {
      severityFilters.value.push(severity)
    }
    pageNum.value = 1
  }

  const toggleSystemKeyFilter = (systemKey: string) => {
    const idx = systemKeyFilters.value.indexOf(systemKey)
    if (idx >= 0) {
      systemKeyFilters.value.splice(idx, 1)
    } else {
      systemKeyFilters.value.push(systemKey)
    }
    pageNum.value = 1
  }

  const toggleAlertNameFilter = (alertname: string) => {
    const idx = alertnameFilters.value.indexOf(alertname)
    if (idx >= 0) {
      alertnameFilters.value.splice(idx, 1)
    } else {
      alertnameFilters.value.push(alertname)
    }
    pageNum.value = 1
  }

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
    resetFilters,
    areDefaultFiltersApplied,
    toggleStatusFilter,
    toggleSeverityFilter,
    toggleSystemKeyFilter,
    toggleAlertNameFilter,
  }
})
