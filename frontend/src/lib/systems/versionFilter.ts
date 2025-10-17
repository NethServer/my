//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const VERSION_FILTER_KEY = 'versionFilter'
export const VERSION_FILTER_PATH = 'filters/versions'

interface VersionFilterResponse {
  code: number
  message: string
  data: {
    versions: string[]
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
