//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'

export const IMPERSONATION_CONSENT_KEY = 'impersonationConsent'
const IMPERSONATE_PATH = 'impersonate'
const CONSENT_PATH = `${IMPERSONATE_PATH}/consent`

interface ImpersonationConsentResponse {
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

interface ImpersonationStatusResponse {
  code: number
  message: string
  data: {
    is_impersonating: boolean
  }
}

export const PostConsentSchema = v.object({
  duration_hours: v.pipe(
    v.number(),
    v.integer('account.impersonation_consent_duration_integer'),
    v.minValue(1, 'account.impersonation_consent_duration_minimum'),
    v.maxValue(168, 'account.impersonation_consent_duration_maximum'),
  ),
})

export type PostConsent = v.InferOutput<typeof PostConsentSchema>

export const getConsent = () => {
  const loginStore = useLoginStore()

  return axios
    .get<ImpersonationConsentResponse>(`${API_URL}/${CONSENT_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const postConsent = (consent: PostConsent) => {
  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/${CONSENT_PATH}`, consent, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteConsent = () => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/${CONSENT_PATH}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const getImpersonationStatus = () => {
  const loginStore = useLoginStore()

  return axios
    .get<ImpersonationStatusResponse>(`${API_URL}/${IMPERSONATE_PATH}/status`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
    .catch((error) => {
      console.error('Cannot get impersonation status:', error)
      throw error
    })
}
