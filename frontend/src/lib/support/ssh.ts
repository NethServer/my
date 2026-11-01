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

// The command an operator pastes to open a shell: ssh tunnelled in a
// WebSocket by websocat, through the proxy that serves this page. With a
// ProxyCommand ssh never dials the host: "my" only names the known_hosts
// entry, the same one an operator's "Host my" config block uses.
export const sshCommand = (systemKey: string, nodeId?: string | null): string => {
  const url = `wss://${window.location.host}/ssh-gateway/api/ssh`
  const user = nodeId ? `${systemKey}.${nodeId}` : systemKey
  return `ssh -o ProxyCommand='websocat --binary ${url}' ${user}@my`
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
