//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { MIN_SEARCH_LENGTH } from '@/lib/common'
import { DEFAULT_PAGE_SIZE, loadPageSizeFromStorage } from '@/lib/tablePageSize'
import { useLoginStore } from '@/stores/login'
import { defineQuery, useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { ref, watch } from 'vue'
import type { NeDropdownFilterV2Option } from '@nethesis/vue-components'
import {
  APPLICATIONS_KEY,
  APPLICATIONS_TABLE_ID,
  getApplications,
  type Application,
} from '@/lib/applications/applications'

export const useApplications = defineQuery(() => {
  const loginStore = useLoginStore()
  const pageNum = ref(1)
  const pageSize = ref(DEFAULT_PAGE_SIZE)
  const textFilter = ref('')
  const debouncedTextFilter = ref('')
  const typeFilter = ref<NeDropdownFilterV2Option[]>([])
  const versionFilter = ref<NeDropdownFilterV2Option[]>([])
  const systemFilter = ref<NeDropdownFilterV2Option[]>([])
  const organizationFilter = ref<NeDropdownFilterV2Option[]>([])
  // when true, the applications of every company in the hierarchy of the
  // selected organization are shown (organizationFilter holds that single
  // organization)
  const includeHierarchy = ref(false)
  const sortBy = ref<keyof Application>('display_name')
  const sortDescending = ref(false)

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [
      APPLICATIONS_KEY,
      {
        pageNum: pageNum.value,
        pageSize: pageSize.value,
        textFilter: debouncedTextFilter.value,
        typeFilter: typeFilter.value.map((o) => o.id),
        versionFilter: versionFilter.value.map((o) => o.id),
        systemFilter: systemFilter.value.map((o) => o.id),
        organizationFilter: organizationFilter.value.map((o) => o.id),
        includeHierarchy: includeHierarchy.value,
        sortBy: sortBy.value,
        sortDirection: sortDescending.value,
      },
    ],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getApplications(
        pageNum.value,
        pageSize.value,
        debouncedTextFilter.value,
        typeFilter.value.map((o) => o.id),
        versionFilter.value.map((o) => o.id),
        systemFilter.value.map((o) => o.id),
        organizationFilter.value.map((o) => o.id),
        includeHierarchy.value,
        sortBy.value,
        sortDescending.value,
      ),
  })

  // load table page size from storage
  watch(
    () => loginStore.userInfo?.email,
    (email) => {
      if (email) {
        pageSize.value = loadPageSizeFromStorage(APPLICATIONS_TABLE_ID)
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
  )

  // reset to first page when version filter changes
  watch(
    () => versionFilter.value,
    () => {
      pageNum.value = 1
    },
  )

  // reset to first page when system filter changes
  watch(
    () => systemFilter.value,
    () => {
      pageNum.value = 1
    },
  )

  // the organization hierarchy mode is scoped to; lets us tell a genuine user
  // change apart from OrganizationDropdownFilter re-emitting the same selection
  // as a fresh array on mount (which must not exit hierarchy mode)
  const hierarchyOrgId = ref<string | null>(null)

  // watch the selected org ids by value (not the array reference): reset to the
  // first page when the selection changes, and exit hierarchy mode whenever the
  // selection moves away from the single organization it was applied to
  watch(
    () => organizationFilter.value.map((o) => o.id).join(','),
    (ids) => {
      pageNum.value = 1

      if (includeHierarchy.value && ids !== (hierarchyOrgId.value ?? '')) {
        includeHierarchy.value = false
        hierarchyOrgId.value = null
      }
    },
  )

  // filter applications by the given organization and every company in its hierarchy
  const applyHierarchyFilter = (organization: NeDropdownFilterV2Option) => {
    clearFilters()
    organizationFilter.value = [organization]
    includeHierarchy.value = true
    hierarchyOrgId.value = organization.id
  }

  const clearFilters = () => {
    textFilter.value = ''
    typeFilter.value = []
    versionFilter.value = []
    systemFilter.value = []
    organizationFilter.value = []
    includeHierarchy.value = false
    hierarchyOrgId.value = null
  }

  return {
    ...rest,
    state,
    asyncStatus,
    pageNum,
    pageSize,
    textFilter,
    typeFilter,
    versionFilter,
    systemFilter,
    organizationFilter,
    includeHierarchy,
    debouncedTextFilter,
    sortBy,
    sortDescending,
    applyHierarchyFilter,
    clearFilters,
  }
})
