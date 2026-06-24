//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getSystems, type System } from '@/lib/systems/systems'
import { useLoginStore } from '@/stores/login'
import { useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { computed, ref, watch } from 'vue'
import type { FilterOption } from '@nethesis/vue-components'
import { OPTIONS_PAGE_SIZE } from '@/lib/common'

const SYSTEMS_SEARCH_KEY = 'systemsSearch'

export function useSystemFilter(idField: 'system_key' | 'id' = 'system_key') {
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
    key: () => [SYSTEMS_SEARCH_KEY, idField, debouncedSearch.value],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getSystems(1, OPTIONS_PAGE_SIZE, debouncedSearch.value, [], [], [], [], [], 'name', false),
  })

  const options = computed<FilterOption[]>(() => {
    const systems = state.value.data?.systems ?? []
    return systems.map((sys: System) => ({
      id: sys[idField] ?? sys.id,
      label: sys.name,
    }))
  })

  const loading = computed(() => asyncStatus.value === 'loading')

  function onSearch(query: string) {
    searchInput.value = query
  }

  return { options, loading, onSearch }
}
