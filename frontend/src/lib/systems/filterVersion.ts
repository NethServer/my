//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const FILTER_VERSION_KEY = 'filterVersion'
export const FILTER_VERSION_PATH = 'filters/versions'

interface FilterVersionResponse {
  code: number
  message: string
  data: {
    versions: string[]
  }
}

export const getFilterVersion = () => {
  const loginStore = useLoginStore()

  return axios
    .get<FilterVersionResponse>(`${API_URL}/${FILTER_VERSION_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
