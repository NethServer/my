//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { MIN_SEARCH_LENGTH } from '@/lib/common'
import { canReadSystems } from '@/lib/permissions'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import {
  getSystems,
  SYSTEMS_KEY,
  SYSTEMS_TABLE_ID,
  type System,
  type SystemStatus,
} from '@/lib/systems/systems'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { ref, watch } from 'vue'
import { computed } from 'vue'

export const useSystems = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const textFilter = ref('')
  const debouncedTextFilter = ref('')
  const productFilter = ref<string[]>([])
  const createdByFilter = ref<string[]>([])
  const versionFilter = ref<string[]>([])
  const statusFilter = ref<SystemStatus[]>(['online', 'offline', 'unknown', 'suspended'])
  const organizationFilter = ref<string[]>([])
  const sortBy = ref<keyof System>('name')
  const sortDescending = ref(false)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      SYSTEMS_KEY,
      {
        pageNum: pageNum.value,
        pageSize: pageSize.value,
        textFilter: debouncedTextFilter.value,
        productFilter: productFilter.value,
        createdByFilter: createdByFilter.value,
        versionFilter: versionFilter.value,
        statusFilter: statusFilter.value,
        organizationFilter: organizationFilter.value,
        sortBy: sortBy.value,
        sortDirection: sortDescending.value,
      },
    ],
    enabled: () => !!loginStore.jwtToken && canReadSystems(),
    query: () =>
      getSystems(
        pageNum.value,
        pageSize.value,
        debouncedTextFilter.value,
        productFilter.value,
        createdByFilter.value,
        versionFilter.value,
        statusFilter.value,
        organizationFilter.value,
        sortBy.value,
        sortDescending.value,
      ),
  })

  const areDefaultFiltersApplied = computed(() => {
    return (
      !debouncedTextFilter.value &&
      productFilter.value.length === 0 &&
      versionFilter.value.length === 0 &&
      createdByFilter.value.length === 0 &&
      organizationFilter.value.length === 0 &&
      statusFilter.value.length === 4 &&
      statusFilter.value.includes('online') &&
      statusFilter.value.includes('offline') &&
      statusFilter.value.includes('unknown') &&
      statusFilter.value.includes('suspended') &&
      !statusFilter.value.includes('deleted')
    )
  })

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(SYSTEMS_TABLE_ID)
      }
    },
    { immediate: true },
  )

  watch(
    () => textFilter.value,
    useDebounceFn(() => {
      // debounce and ignore if text filter is too short
      if (textFilter.value.length === 0 || textFilter.value.length >= MIN_SEARCH_LENGTH) {
        debouncedTextFilter.value = textFilter.value

        // reset to first page when filter changes
        pageNum.value = 1
      }
    }, 500),
  )

  // reset to first page when page size changes
  watch(
    () => pageSize.value,
    () => {
      pageNum.value = 1
    },
  )

  const resetFilters = () => {
    textFilter.value = ''
    productFilter.value = []
    versionFilter.value = []
    createdByFilter.value = []
    statusFilter.value = ['online', 'offline', 'unknown', 'suspended']
    organizationFilter.value = []
  }

  return {
    ...rest,
    state,
    asyncStatus,
    pageNum,
    pageSize,
    textFilter,
    productFilter,
    createdByFilter,
    versionFilter,
    statusFilter,
    organizationFilter,
    debouncedTextFilter,
    sortBy,
    sortDescending,
    areDefaultFiltersApplied,
    resetFilters,
  }
})
