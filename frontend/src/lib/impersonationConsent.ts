//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'

export const IMPERSONATION_CONSENT_KEY = 'impersonationConsent'
export const CONSENT_DURATION_HOURS = 24
const CONSENT_PATH = 'impersonate/consent'

interface ConsentResponse {
  code: number
  message: string
  data: {
    consent: {
      id: string
      user_id: string
      expires_at: string
      max_duration_minutes: number
      created_at: string
      active: boolean
    }
  }
}

export const getConsent = () => {
  const loginStore = useLoginStore()

  return axios
    .get<ConsentResponse>(`${API_URL}/${CONSENT_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const postConsent = (durationHours: number) => {
  const loginStore = useLoginStore()

  return axios.post(
    `${API_URL}/${CONSENT_PATH}`,
    { duration_hours: durationHours },
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const deleteConsent = () => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/${CONSENT_PATH}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}
