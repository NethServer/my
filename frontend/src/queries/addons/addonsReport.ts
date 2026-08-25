//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { defineQuery, useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { ref, watch } from 'vue'
import {
  ADDONS_REPORT_KEY,
  ADDONS_REPORT_ORGANIZATIONS_KEY,
  ADDONS_REPORT_ORGANIZATIONS_TABLE_ID,
  ADDONS_REPORT_TIERS_KEY,
  ADDONS_REPORT_TIERS_TABLE_ID,
  getAddonReport,
  getAddonReportOrganizations,
  getAddonReportTiers,
} from '@/lib/addons/addonsReport'
import { MIN_SEARCH_LENGTH } from '@/lib/common'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'

// The aggregates arrive whole: totals, the per-add-on breakdown, the renewal
// distribution and the activation trend are one response, with no parameters
// to give it.
export const useAddonReport = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [ADDONS_REPORT_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getAddonReport(),
  })

  return { ...rest, state, asyncStatus }
})

// The per-organization and per-tier slices page and search on the server, so
// each keeps its own paging state and rebuilds its key from it. Written twice
// rather than shared: the two differ in table id and in what the search box
// matches, and a factory would hide that behind an argument.
export const useAddonReportOrganizations = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const textFilter = ref('')
  const debouncedTextFilter = ref('')

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ADDONS_REPORT_ORGANIZATIONS_KEY,
      { pageNum: pageNum.value, pageSize: pageSize.value, textFilter: debouncedTextFilter.value },
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getAddonReportOrganizations(pageNum.value, pageSize.value, debouncedTextFilter.value),
  })

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(ADDONS_REPORT_ORGANIZATIONS_TABLE_ID)
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

  return { ...rest, state, asyncStatus, pageNum, pageSize, textFilter }
})

export const useAddonReportTiers = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const textFilter = ref('')
  const debouncedTextFilter = ref('')

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      ADDONS_REPORT_TIERS_KEY,
      { pageNum: pageNum.value, pageSize: pageSize.value, textFilter: debouncedTextFilter.value },
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () => getAddonReportTiers(pageNum.value, pageSize.value, debouncedTextFilter.value),
  })

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(ADDONS_REPORT_TIERS_TABLE_ID)
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

  return { ...rest, state, asyncStatus, pageNum, pageSize, textFilter }
})
