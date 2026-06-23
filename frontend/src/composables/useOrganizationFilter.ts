//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  ORGANIZATIONS_SEARCH_KEY,
  searchOrganizations,
} from '@/lib/organizations/searchOrganizations'
import { useLoginStore } from '@/stores/login'
import { useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { computed, ref, watch } from 'vue'
import type { FilterOption } from '@nethesis/vue-components'

export function useOrganizationFilter() {
  const loginStore = useLoginStore()
  const searchInput = ref('')
  const debouncedSearch = ref('')

  watch(
    () => searchInput.value,
    useDebounceFn(() => {
      debouncedSearch.value = searchInput.value
    }, 300),
  )

  const { state, asyncStatus } = useQuery({
    key: () => [ORGANIZATIONS_SEARCH_KEY, debouncedSearch.value],
    enabled: () => !!loginStore.jwtToken,
    query: () => searchOrganizations(debouncedSearch.value),
  })

  const options = computed<FilterOption[]>(() => {
    const orgs = state.value.data ?? []
    return orgs.map((org) => ({
      id: org.logto_id,
      label: org.name,
    }))
  })

  const loading = computed(() => asyncStatus.value === 'loading')

  function onSearch(query: string) {
    searchInput.value = query
  }

  return { options, loading, onSearch, currentSearch: searchInput }
}
