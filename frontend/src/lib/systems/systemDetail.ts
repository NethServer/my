//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { System } from './systems'

interface SystemDetailResponse {
  code: number
  message: string
  data: System
}

export const getSystemDetail = (systemId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<SystemDetailResponse>(`${API_URL}/systems/${systemId}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
