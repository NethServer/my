//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { defineQuery, useQuery } from '@pinia/colada'
import { useRoute } from 'vue-router'
import {
  AVAILABLE_ADDONS_KEY,
  SYSTEM_ADDONS_KEY,
  getAvailableAddons,
  getSystemAddons,
} from '@/lib/addons/systemAddons'
import { APPLICATIONS_KEY, getApplications } from '@/lib/applications/applications'
import { useLoginStore } from '@/stores/login'

// Every application instance of the system in one request: the add-ons panel
// needs the whole set to build its rows, and a cluster never has enough
// instances to page through.
const SYSTEM_APPLICATIONS_PAGE_SIZE = 200

// The licenses and the instances they go to change behind the page — an
// inventory frees the license of an instance that left the cluster — and the
// two are read together: both are fetched again whenever the page is opened or
// the tab regains focus, so neither shows a state the other has moved past.
const ALWAYS_FRESH = { refetchOnMount: 'always', refetchOnWindowFocus: 'always' } as const

export const useSystemAddons = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [SYSTEM_ADDONS_KEY, route.params.systemId as string],
    enabled: () => !!loginStore.jwtToken && !!route.params.systemId,
    query: () => getSystemAddons(route.params.systemId as string),
    ...ALWAYS_FRESH,
  })

  return { ...rest, state, asyncStatus }
})

export const useAvailableAddons = defineQuery(() => {
  const loginStore = useLoginStore()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [AVAILABLE_ADDONS_KEY],
    enabled: () => !!loginStore.jwtToken,
    query: () => getAvailableAddons(),
  })

  return { ...rest, state, asyncStatus }
})

export const useSystemApplications = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [APPLICATIONS_KEY, 'system', route.params.systemId as string],
    enabled: () => !!loginStore.jwtToken && !!route.params.systemId,
    query: () =>
      getApplications(
        1,
        SYSTEM_APPLICATIONS_PAGE_SIZE,
        '',
        [],
        [],
        [route.params.systemId as string],
        [],
        'module_id',
        false,
      ),
    ...ALWAYS_FRESH,
  })

  return { ...rest, state, asyncStatus }
})
