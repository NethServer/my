//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { FilterOption } from '@nethesis/vue-components'

export const APPLICATION_VERSION_FILTER_KEY = 'applicationVersionFilter'

const APPLICATION_VERSION_FILTER_PATH = 'filters/applications/versions'

export interface ApplicationVersions {
  application: string
  name: string
  versions: string[]
}

interface VersionFilterResponse {
  code: number
  message: string
  data: {
    versions: ApplicationVersions[]
  }
}

export const getVersionFilter = () => {
  const loginStore = useLoginStore()

  return axios
    .get<VersionFilterResponse>(`${API_URL}/${APPLICATION_VERSION_FILTER_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const buildVersionFilterOptions = (applicationVersions: ApplicationVersions[]) => {
  const options: FilterOption[] = []

  applicationVersions.forEach((av) => {
    const appName = av.name

    av.versions.forEach((appAndVersion) => {
      // split application and version
      const [, version] = appAndVersion.split(':')

      if (appName && version) {
        options.push({
          id: appAndVersion,
          label: `${appName} ${version}`,
        })
      }
    })
  })
  return options
}
