//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

// An SSH access request opened by the SSH gateway, waiting for the operator
// who copied the command
export interface SSHAccessRequest {
  system_id: string
  system_name: string
  system_type: string
  system_key: string
  organization_name: string
  node_id: string
  client_ip: string
  fingerprint: string
  created_at: string
  has_session: boolean
}

interface SSHTokenResponse {
  data: { token: string; system_key: string; node_id: string; expires_in: number }
}

// what may appear in the copied command: anything else could inject into the
// operator's shell when pasted
const SYSTEM_KEY = /^[A-Za-z0-9-]{1,128}$/
const NODE_ID = /^[0-9]{1,10}$/
const TOKEN = /^[0-9a-f]{32}$/

// The command an operator pastes to open a shell: ssh tunnelled in a
// WebSocket by websocat, through the proxy that serves this page. The user
// carries a one-time token bound to this operator, so only they can approve
// the access it requests. With a ProxyCommand ssh never dials the host: "my"
// only names the known_hosts entry.
export const sshCommand = (systemKey: string, nodeId: string, token: string): string => {
  if (!SYSTEM_KEY.test(systemKey) || (nodeId && !NODE_ID.test(nodeId)) || !TOKEN.test(token)) {
    throw new Error('invalid ssh target')
  }
  const url = `wss://${window.location.host}/ssh-gateway/api/ssh`
  const target = nodeId ? `${systemKey}.${nodeId}` : systemKey
  return `ssh -o ProxyCommand='websocat --binary ${url}' ${target}+${token}@my`
}

// Mints the token and builds the command for a system's node
export const createSSHCommand = (systemId: string, nodeId?: string | null): Promise<string> =>
  axios
    .post<SSHTokenResponse>(
      `${API_URL}/support-ssh/tokens`,
      { system_id: systemId, node_id: nodeId || '' },
      { headers: authHeaders() },
    )
    .then((res) => sshCommand(res.data.data.system_key, res.data.data.node_id, res.data.data.token))

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
