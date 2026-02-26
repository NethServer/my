//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { Application } from './applications'

interface ApplicationDetailResponse {
  code: number
  message: string
  data: Application
}

export const getApplicationDetail = (applicationId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<ApplicationDetailResponse>(`${API_URL}/applications/${applicationId}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
