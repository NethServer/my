//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { FilterOption, FilterOptionGroup } from '@nethesis/vue-components'
import { getProductName } from './systems'

export const SYSTEM_FILTERS_KEY = 'systemFilters'

const SYSTEM_FILTERS_PATH = 'filters/systems'

export interface ProductVersions {
  product: string
  versions: string[]
}

export interface CreatedByItem {
  user_id: string
  name: string
}

export interface OrganizationItem {
  id: string // logto_id
  name: string
}

export interface SystemFiltersData {
  products: string[]
  versions: ProductVersions[]
  created_by: CreatedByItem[]
  organizations: OrganizationItem[]
}

interface SystemFiltersResponse {
  code: number
  message: string
  data: SystemFiltersData
}

export const getSystemFilters = () => {
  const loginStore = useLoginStore()

  return axios
    .get<SystemFiltersResponse>(`${API_URL}/${SYSTEM_FILTERS_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

/**
 * Builds version filter options from ProductVersions array
 * Used by components to format version data for dropdown display
 */
export const buildVersionFilterOptions = (productVersions: ProductVersions[]) => {
  const optionGroups: FilterOptionGroup[] = []

  productVersions.forEach((pv) => {
    const options: FilterOption[] = []

    optionGroups.push({
      group: getProductName(pv.product),
      options,
    })

    pv.versions.forEach((productAndVersion) => {
      // split product and version
      const [, version] = productAndVersion.split(':')

      if (version) {
        options.push({
          id: productAndVersion,
          label: version,
        })
      }
    })
  })
  return optionGroups
}
