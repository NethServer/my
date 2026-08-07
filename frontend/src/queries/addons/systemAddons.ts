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

export const useSystemAddons = defineQuery(() => {
  const loginStore = useLoginStore()
  const route = useRoute()

  const { state, asyncStatus, ...rest } = useQuery({
    key: () => [SYSTEM_ADDONS_KEY, route.params.systemId as string],
    enabled: () => !!loginStore.jwtToken && !!route.params.systemId,
    query: () => getSystemAddons(route.params.systemId as string),
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
        false,
        'module_id',
        false,
      ),
  })

  return { ...rest, state, asyncStatus }
})
