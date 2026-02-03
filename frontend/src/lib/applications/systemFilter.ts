//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const APPLICATION_SYSTEM_FILTER_KEY = 'applicationSystemFilter'

const APPLICATION_SYSTEM_FILTER_PATH = 'filters/applications/systems'

interface SystemFilterResponse {
  code: number
  message: string
  data: ApplicationSystem[]
}

interface ApplicationSystem {
  id: string
  name: string
}

export const getSystemFilter = () => {
  const loginStore = useLoginStore()

  return axios
    .get<SystemFilterResponse>(`${API_URL}/${APPLICATION_SYSTEM_FILTER_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
