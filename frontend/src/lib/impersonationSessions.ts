//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import { getQueryStringParams, type Pagination } from './common'
import { IMPERSONATE_PATH } from './impersonation'

export const SESSIONS_KEY = 'impersonationSessions'
export const SESSIONS_TABLE_ID = 'impersonationSessionsTable'
const SESSIONS_PATH = `${IMPERSONATE_PATH}/sessions`

export interface Session {
  session_id: string
  impersonator_user_id: string
  impersonated_user_id: string
  impersonator_username: string
  impersonated_username: string
  start_time: string
  end_time: string
  duration_minutes: number
  action_count: number
  status: 'active' | 'completed'
}

interface SessionsResponse {
  code: number
  message: string
  data: {
    sessions: Session[]
    pagination: Pagination
  }
}

export const getSessions = (
  pageNum: number,
  pageSize: number,
  sortBy: string,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParams(pageNum, pageSize, null, sortBy, sortDescending)

  return axios
    .get<SessionsResponse>(`${API_URL}/${SESSIONS_PATH}?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
