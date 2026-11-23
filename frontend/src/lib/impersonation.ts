//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore, type UserInfo } from '@/stores/login'

export const IMPERSONATE_PATH = 'impersonate'

interface ImpersonateResponse {
  code: number
  message: string
  data: {
    is_impersonating: boolean
    impersonated_user: UserInfo
    impersonator: UserInfo
    session_id: string
    token: string
    expires_in: number
    expires_at: string
  }
}

interface DeleteImpersonateResponse {
  code: number
  message: string
  data: {
    expires_in: number
    token: string
    refresh_token: string
    user: UserInfo
  }
}

export const getImpersonationStatus = () => {
  const loginStore = useLoginStore()

  return axios
    .get<ImpersonateResponse>(`${API_URL}/${IMPERSONATE_PATH}/status`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
    .catch((error) => {
      console.error('Cannot get impersonation status:', error)
      throw error
    })
}

export const postImpersonate = (userId: string) => {
  const loginStore = useLoginStore()

  return axios
    .post<ImpersonateResponse>(
      `${API_URL}/${IMPERSONATE_PATH}`,
      { user_id: userId },
      {
        headers: {
          Authorization: `Bearer ${loginStore.jwtToken}`,
        },
      },
    )
    .then((res) => res.data.data)
    .catch((error) => {
      console.error('Cannot post impersonate:', error)
      throw error
    })
}

// restarts the current session for a full duration: a new token, same session
export const postExtendImpersonation = () => {
  const loginStore = useLoginStore()

  return axios
    .post<ImpersonateResponse>(`${API_URL}/${IMPERSONATE_PATH}/extend`, null, {
      headers: {
        Authorization: `Bearer ${loginStore.jwtToken}`,
      },
    })
    .then((res) => res.data.data)
    .catch((error) => {
      console.error('Cannot extend impersonation:', error)
      throw error
    })
}

export const deleteImpersonate = () => {
  const loginStore = useLoginStore()

  return axios
    .delete<DeleteImpersonateResponse>(`${API_URL}/${IMPERSONATE_PATH}`, {
      headers: {
        Authorization: `Bearer ${loginStore.jwtToken}`,
      },
    })
    .then((res) => res.data.data)
    .catch((error) => {
      console.error('Cannot delete impersonate:', error)
      throw error
    })
}
