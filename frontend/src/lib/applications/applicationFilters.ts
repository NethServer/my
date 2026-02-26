//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { FilterOption, FilterOptionGroup } from '@nethesis/vue-components'

export const APPLICATION_FILTERS_KEY = 'applicationFilters'

const APPLICATION_FILTERS_PATH = 'filters/applications'

export interface ApplicationType {
  instance_of: string
  name: string
  count: number
}

export interface ApplicationVersions {
  application: string
  name: string
  versions: string[]
}

export interface SystemSummary {
  id: string
  name: string
}

export interface OrganizationSummary {
  id: string
  logto_id: string
  name: string
  type?: string
  description?: string
}

export interface ApplicationFiltersData {
  types: ApplicationType[]
  versions: ApplicationVersions[]
  systems: SystemSummary[]
  organizations: OrganizationSummary[]
}

interface ApplicationFiltersResponse {
  code: number
  message: string
  data: ApplicationFiltersData
}

export const getApplicationFilters = () => {
  const loginStore = useLoginStore()

  return axios
    .get<ApplicationFiltersResponse>(`${API_URL}/${APPLICATION_FILTERS_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

/**
 * Builds version filter options from ApplicationVersions array
 * Used by components to format version data for dropdown display
 */
export const buildVersionFilterOptions = (applicationVersions: ApplicationVersions[]) => {
  const optionGroups: FilterOptionGroup[] = []
  const seen = new Set<string>()

  applicationVersions.forEach((av) => {
    const appName = av.name
    const options: FilterOption[] = []

    optionGroups.push({
      group: appName,
      options,
    })

    av.versions.forEach((appAndVersion) => {
      // split application and version
      const [, version] = appAndVersion.split(':')

      if (appName && version && !seen.has(appAndVersion)) {
        seen.add(appAndVersion)
        options.push({
          id: appAndVersion,
          label: version,
        })
      }
    })
  })
  return optionGroups
}
