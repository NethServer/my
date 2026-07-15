//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { getUsers, type User } from '@/lib/users/users'
import { useLoginStore } from '@/stores/login'
import { useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { computed, ref, watch } from 'vue'
import type { NeDropdownFilterV2Option } from '@nethesis/vue-components'
import { OPTIONS_PAGE_SIZE } from '@/lib/common'

const ASSIGNEES_SEARCH_KEY = 'assigneesSearch'

// Provides the assignee filter options from the first page of GET /users,
// with server-side (external) filtering driven by the dropdown search input.
export function useAssigneeFilter() {
  const loginStore = useLoginStore()
  const searchInput = ref('')
  const debouncedSearch = ref('')
  const currentSearch = computed(() => debouncedSearch.value)

  watch(
    () => searchInput.value,
    useDebounceFn(() => {
      debouncedSearch.value = searchInput.value
    }, 300),
  )

  const { state, asyncStatus } = useQuery({
    key: () => [ASSIGNEES_SEARCH_KEY, debouncedSearch.value],
    enabled: () => !!loginStore.jwtToken,
    query: () =>
      getUsers(1, OPTIONS_PAGE_SIZE, debouncedSearch.value, [], [], [], [], 'name', false),
  })

  // The backend stores the assignee id as the Logto id when available, so we
  // key options by logto_id and skip users that don't have one (they can't be
  // matched against alert assignments).
  const options = computed<NeDropdownFilterV2Option[]>(() => {
    const users = state.value.data?.users ?? []
    return users
      .filter((u: User) => !!u.logto_id)
      .map((u: User) => ({ id: u.logto_id as string, label: u.name }))
  })

  const loading = computed(() => asyncStatus.value === 'loading')

  function onSearch(query: string) {
    searchInput.value = query
  }

  return { options, loading, onSearch, currentSearch }
}
