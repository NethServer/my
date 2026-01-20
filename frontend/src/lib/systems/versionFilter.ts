//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { FilterOption } from '@nethesis/vue-components'
import { getProductName } from './systems'

export const VERSION_FILTER_KEY = 'versionFilter'
export const VERSION_FILTER_PATH = 'filters/systems/versions'

export interface ProductVersions {
  product: string
  versions: string[]
}

interface VersionFilterResponse {
  code: number
  message: string
  data: {
    versions: ProductVersions[]
  }
}

export const getVersionFilter = () => {
  const loginStore = useLoginStore()

  return axios
    .get<VersionFilterResponse>(`${API_URL}/${VERSION_FILTER_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const buildVersionFilterOptions = (productVersions: ProductVersions[]) => {
  const options: FilterOption[] = []

  productVersions.forEach((pv) => {
    pv.versions.forEach((productAndVersion) => {
      // split product and version
      const [product, version] = productAndVersion.split(':')

      if (product && version) {
        options.push({
          id: productAndVersion,
          label: `${getProductName(product)} ${version}`,
        })
      }
    })
  })
  return options
}
