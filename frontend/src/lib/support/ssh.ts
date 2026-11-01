//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

// An SSH access request opened by the SSH gateway, waiting for an operator
export interface SSHAccessRequest {
  system_id: string
  system_name: string
  system_type: string
  system_key: string
  node_id: string
  client_ip: string
  user_code: string
  created_at: string
  has_session: boolean
}

const authHeaders = () => ({ Authorization: `Bearer ${useLoginStore().jwtToken}` })

export const getSSHAccessRequest = (code: string): Promise<SSHAccessRequest> =>
  axios
    .get<{ data: SSHAccessRequest }>(
      `${API_URL}/support-ssh/requests/${encodeURIComponent(code)}`,
      { headers: authHeaders() },
    )
    .then((res) => res.data.data)

export const approveSSHAccessRequest = (code: string): Promise<void> =>
  axios
    .post(`${API_URL}/support-ssh/requests/${encodeURIComponent(code)}/approve`, null, {
      headers: authHeaders(),
    })
    .then(() => undefined)

export const denySSHAccessRequest = (code: string): Promise<void> =>
  axios
    .post(`${API_URL}/support-ssh/requests/${encodeURIComponent(code)}/deny`, null, {
      headers: authHeaders(),
    })
    .then(() => undefined)
