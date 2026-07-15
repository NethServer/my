//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import {
  ORGANIZATIONS_SEARCH_KEY,
  searchOrganizations,
} from '@/lib/organizations/searchOrganizations'
import { useLoginStore } from '@/stores/login'
import { useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { computed, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import type { FilterOption } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'

/**
 * Server-searched organization options for comboboxes/filters.
 *
 * @param enabled gates the underlying request; pass the field's visibility so
 *   the query doesn't fire while the host drawer is closed. Defaults to always on.
 */
export function useOrganizationFilter(enabled?: MaybeRefOrGetter<boolean>) {
  const { t } = useI18n()
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
    enabled: () => !!loginStore.jwtToken && toValue(enabled ?? true),
    query: () => searchOrganizations(debouncedSearch.value),
  })

  const organizations = computed(() => state.value.data ?? [])

  const options = computed<FilterOption[]>(() =>
    organizations.value.map((org) => ({
      id: org.logto_id,
      label: org.name,
      description: t(`organizations.${org.type}`),
    })),
  )

  const loading = computed(() => asyncStatus.value === 'loading')

  function onSearch(query: string) {
    searchInput.value = query
  }

  return { options, organizations, loading, onSearch, currentSearch: searchInput }
}

/**
 * Whether the current user has at least one organization it could attribute a
 * new entity to — an org of one of `allowedTypes`, other than its own. Lets a
 * parent decide (via v-if) whether to show the "created on behalf of" field,
 * instead of the combobox hiding itself. Never searches, so the result reflects
 * the full eligible set and stays stable while the user types in the combobox.
 *
 * @param enabled gates the request (e.g. drawer open AND caller may attribute).
 */
export function useHasAttributableOrganizations(
  allowedTypes: MaybeRefOrGetter<string[]>,
  enabled?: MaybeRefOrGetter<boolean>,
) {
  const loginStore = useLoginStore()
  const { organizations } = useOrganizationFilter(enabled)

  return computed(() =>
    organizations.value.some(
      (org) =>
        toValue(allowedTypes).includes(org.type) &&
        org.logto_id !== loginStore.userInfo?.organization_id,
    ),
  )
}
