//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { MIN_SEARCH_LENGTH } from '@/lib/common'
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
import type { NeDropdownFilterV2Option } from '@nethesis/vue-components'

export const useSystems = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const textFilter = ref('')
  const debouncedTextFilter = ref('')
  const productFilter = ref<NeDropdownFilterV2Option[]>([])
  const versionFilter = ref<NeDropdownFilterV2Option[]>([])
  const statusFilter = ref<NeDropdownFilterV2Option[]>([
    { id: 'active', label: 'active' },
    { id: 'inactive', label: 'inactive' },
    { id: 'unknown', label: 'unknown' },
    { id: 'suspended', label: 'suspended' },
  ])
  const organizationFilter = ref<NeDropdownFilterV2Option[]>([])
  // "Managed by": the systems whose company sits anywhere in the subtree of
  // these organizations (ANDed with organizationFilter)
  const parentOrganizationFilter = ref<NeDropdownFilterV2Option[]>([])
  const addonFilter = ref<NeDropdownFilterV2Option[]>([])
  const sortBy = ref<keyof System>('name')
  const sortDescending = ref(false)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      SYSTEMS_KEY,
      {
        pageNum: pageNum.value,
        pageSize: pageSize.value,
        textFilter: debouncedTextFilter.value,
        productFilter: productFilter.value.map((o) => o.id),
        versionFilter: versionFilter.value.map((o) => o.id),
        statusFilter: statusFilter.value.map((o) => o.id),
        organizationFilter: organizationFilter.value.map((o) => o.id),
        parentOrganizationFilter: parentOrganizationFilter.value.map((o) => o.id),
        addonFilter: addonFilter.value.map((o) => o.id),
        sortBy: sortBy.value,
        sortDirection: sortDescending.value,
      },
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getSystems(
        pageNum.value,
        pageSize.value,
        debouncedTextFilter.value,
        productFilter.value.map((o) => o.id),
        [], // created by: not filtered from the tables
        versionFilter.value.map((o) => o.id),
        statusFilter.value.map((o) => o.id) as SystemStatus[],
        organizationFilter.value.map((o) => o.id),
        addonFilter.value.map((o) => o.id),
        sortBy.value,
        sortDescending.value,
        parentOrganizationFilter.value.map((o) => o.id),
      ),
  })

  const areDefaultFiltersApplied = computed(() => {
    return (
      !debouncedTextFilter.value &&
      productFilter.value.length === 0 &&
      versionFilter.value.length === 0 &&
      organizationFilter.value.length === 0 &&
      parentOrganizationFilter.value.length === 0 &&
      addonFilter.value.length === 0 &&
      statusFilter.value.length === 4 &&
      statusFilter.value.some((o) => o.id === 'active') &&
      statusFilter.value.some((o) => o.id === 'inactive') &&
      statusFilter.value.some((o) => o.id === 'unknown') &&
      statusFilter.value.some((o) => o.id === 'suspended') &&
      !statusFilter.value.some((o) => o.id === 'deleted')
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

  // reset to first page when status filter changes
  watch(
    () => statusFilter.value,
    () => {
      pageNum.value = 1
    },
  )

  // reset to first page when product filter changes
  watch(
    () => productFilter.value,
    () => {
      pageNum.value = 1
    },
  )

  // reset to first page when version filter changes
  watch(
    () => versionFilter.value,
    () => {
      pageNum.value = 1
    },
  )

  // reset to first page when parent company filter changes
  watch(
    () => parentOrganizationFilter.value,
    () => {
      pageNum.value = 1
    },
  )

  // reset to first page when add-on filter changes
  watch(
    () => addonFilter.value,
    () => {
      pageNum.value = 1
    },
  )

  // watch the selected org ids by value (not the array reference): reset to the
  // first page when the assigned company selection changes
  watch(
    () => organizationFilter.value.map((o) => o.id).join(','),
    () => {
      pageNum.value = 1
    },
  )

  const resetFilters = () => {
    textFilter.value = ''
    productFilter.value = []
    versionFilter.value = []
    organizationFilter.value = []
    parentOrganizationFilter.value = []
    addonFilter.value = []
    resetStatusFilter()
  }

  const resetStatusFilter = () => {
    statusFilter.value = [
      { id: 'active', label: 'active' },
      { id: 'inactive', label: 'inactive' },
      { id: 'unknown', label: 'unknown' },
      { id: 'suspended', label: 'suspended' },
    ]
  }

  return {
    ...rest,
    state,
    asyncStatus,
    pageNum,
    pageSize,
    textFilter,
    productFilter,
    versionFilter,
    statusFilter,
    organizationFilter,
    parentOrganizationFilter,
    addonFilter,
    debouncedTextFilter,
    sortBy,
    sortDescending,
    areDefaultFiltersApplied,
    resetFilters,
    resetStatusFilter,
  }
})
