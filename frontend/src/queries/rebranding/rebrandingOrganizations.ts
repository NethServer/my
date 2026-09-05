//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { MIN_SEARCH_LENGTH } from '@/lib/common'
import { canReadRebranding } from '@/lib/permissions'
import { REBRANDING_ORGANIZATIONS_KEY, REBRANDING_TABLE_ID } from '@/lib/rebranding/rebranding'
import {
  getRebrandingOrganizations,
  type RebrandingSortBy,
} from '@/lib/rebranding/rebrandingOrganizations'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { computed, ref, watch } from 'vue'
import type { NeDropdownFilterV2Option } from '@nethesis/vue-components'

export const useRebrandingOrganizations = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const textFilter = ref('')
  const debouncedTextFilter = ref('')
  const typeFilter = ref<NeDropdownFilterV2Option[]>([])
  const sortBy = ref<RebrandingSortBy>('name')
  const sortDescending = ref(false)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      REBRANDING_ORGANIZATIONS_KEY,
      {
        pageNum: pageNum.value,
        pageSize: pageSize.value,
        textFilter: debouncedTextFilter.value,
        typeFilter: typeFilter.value.map((o) => o.id),
        sortBy: sortBy.value,
        sortDirection: sortDescending.value,
      },
    ],
    // Without the permission check this 403s, and the axios interceptor turns
    // any 403 into a redirect to /forbidden.
    enabled: () => !!loginStore.jwtToken && canReadRebranding(),
    query: () =>
      getRebrandingOrganizations(
        pageNum.value,
        pageSize.value,
        debouncedTextFilter.value,
        typeFilter.value.map((o) => o.id),
        sortBy.value,
        sortDescending.value,
      ),
  })

  const areDefaultFiltersApplied = computed(
    () => !debouncedTextFilter.value && typeFilter.value.length === 0,
  )

  const resetFilters = () => {
    textFilter.value = ''
    typeFilter.value = []
  }

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(REBRANDING_TABLE_ID)
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

  // reset to first page when type filter changes
  watch(
    () => typeFilter.value,
    () => {
      pageNum.value = 1
    },
    { deep: true },
  )

  return {
    ...rest,
    state,
    asyncStatus,
    pageNum,
    pageSize,
    textFilter,
    debouncedTextFilter,
    typeFilter,
    sortBy,
    sortDescending,
    areDefaultFiltersApplied,
    resetFilters,
  }
})
