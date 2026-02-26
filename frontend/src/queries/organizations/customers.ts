//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { MIN_SEARCH_LENGTH } from '@/lib/common'
import {
  CUSTOMERS_KEY,
  CUSTOMERS_TABLE_ID,
  getCustomers,
  type Customer,
  type CustomerStatus,
} from '@/lib/organizations/customers'
import { canReadCustomers } from '@/lib/permissions'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { computed, ref, watch } from 'vue'

export const useCustomers = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const textFilter = ref('')
  const debouncedTextFilter = ref('')
  const statusFilter = ref<CustomerStatus[]>(['enabled', 'suspended'])
  const sortBy = ref<keyof Customer>('name')
  const sortDescending = ref(false)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      CUSTOMERS_KEY,
      {
        pageNum: pageNum.value,
        pageSize: pageSize.value,
        textFilter: debouncedTextFilter.value,
        statusFilter: statusFilter.value,
        sortBy: sortBy.value,
        sortDirection: sortDescending.value,
      },
    ],
    enabled: () => !!loginStore.jwtToken && canReadCustomers(),
    query: () =>
      getCustomers(
        pageNum.value,
        pageSize.value,
        debouncedTextFilter.value,
        statusFilter.value,
        sortBy.value,
        sortDescending.value,
      ),
  })

  const areDefaultFiltersApplied = computed(() => {
    return (
      !debouncedTextFilter.value &&
      statusFilter.value.length === 2 &&
      statusFilter.value.includes('enabled') &&
      statusFilter.value.includes('suspended') &&
      !statusFilter.value.includes('deleted')
    )
  })

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(CUSTOMERS_TABLE_ID)
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

  // reset to first page when status filter changes
  watch(
    () => statusFilter.value,
    () => {
      pageNum.value = 1
    },
  )

  const resetFilters = () => {
    textFilter.value = ''
    statusFilter.value = ['enabled', 'suspended']
  }

  return {
    ...rest,
    state,
    asyncStatus,
    pageNum,
    pageSize,
    textFilter,
    debouncedTextFilter,
    statusFilter,
    sortBy,
    sortDescending,
    areDefaultFiltersApplied,
    resetFilters,
  }
})
