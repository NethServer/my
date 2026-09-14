//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { OPTIONS_PAGE_SIZE } from '@/lib/common'
import { getOrganizationIcon } from '@/lib/organizations/organizations'
import { canReadRebranding } from '@/lib/permissions'
import { REBRANDING_AVAILABLE_KEY } from '@/lib/rebranding/rebranding'
import { getAvailableRebrandingOrganizations } from '@/lib/rebranding/rebrandingOrganizations'
import { useLoginStore } from '@/stores/login'
import { useQuery } from '@pinia/colada'
import { useDebounceFn } from '@vueuse/core'
import { computed, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import type { NeMultiselectComboboxOption } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'

/**
 * Server-searched options for the "Add companies to rebranding" picker.
 *
 * The endpoint already excludes the organizations that are enabled and the
 * owner organization itself, so whatever comes back can be offered as is. It
 * has no pagination and caps at 200 rows, which is why the search runs on the
 * server rather than over a prefetched list.
 *
 * @param enabled gates the request; pass the drawer's visibility so nothing is
 *   fetched while it is closed. Defaults to always on.
 */
export function useAvailableRebrandingOrganizations(enabled?: MaybeRefOrGetter<boolean>) {
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
    key: () => [REBRANDING_AVAILABLE_KEY, debouncedSearch.value],
    enabled: () => !!loginStore.jwtToken && canReadRebranding() && toValue(enabled ?? true),
    query: () => getAvailableRebrandingOrganizations(debouncedSearch.value, OPTIONS_PAGE_SIZE),
  })

  const organizations = computed(() => state.value.data ?? [])

  // The id is the Logto id, which is what the enable endpoint resolves against:
  // sending the database id answers 400 "organization_ids unknown".
  const options = computed<NeMultiselectComboboxOption[]>(() =>
    organizations.value.map((organization) => ({
      id: organization.logto_id,
      label: organization.name,
      description: t(`organizations.${organization.organization_type}`),
      icon: getOrganizationIcon(organization.organization_type),
    })),
  )

  const loading = computed(() => asyncStatus.value === 'loading')

  function onSearch(query: string) {
    searchInput.value = query
  }

  function resetSearch() {
    searchInput.value = ''
    debouncedSearch.value = ''
  }

  return { options, organizations, loading, onSearch, resetSearch, currentSearch: searchInput }
}
