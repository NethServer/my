//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import { getQueryStringParams, type Pagination } from './common'
import { IMPERSONATE_PATH } from './impersonation'

export const SESSIONS_KEY = 'impersonationSessions'
export const SESSION_AUDIT_KEY = 'impersonationSessionAudit'
export const SESSIONS_TABLE_ID = 'impersonationSessionsTable'
export const SESSION_AUDIT_TABLE_ID = 'impersonationSessionsTable'
const SESSIONS_PATH = `${IMPERSONATE_PATH}/sessions`

export interface Session {
  session_id: string
  impersonator_user_id: string
  impersonated_user_id: string
  impersonator_name: string
  impersonated_name: string
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

interface SessionAuditResponse {
  code: number
  message: string
  data: {
    entries: ImpersonationAuditEntry[]
    pagination: Pagination
    session_id: string
  }
}

interface ImpersonationAuditEntry {
  id: string
  session_id: string
  impersonator_user_id: string
  impersonated_user_id: string
  action_type: string
  api_endpoint: string | null
  http_method: string | null
  request_data: string | null
  response_status: number | null
  response_status_text: string | null
  timestamp: string
  impersonator_username: string
  impersonated_username: string
  impersonator_name: string
  impersonated_name: string
}

export const getSessions = (
  pageNum: number,
  pageSize: number,
  // sortBy: string, ////
  // sortDescending: boolean, ////
) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParams(pageNum, pageSize, null, null, false)

  return axios
    .get<SessionsResponse>(`${API_URL}/${SESSIONS_PATH}?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getSessionAudit = (
  sessionId: string,
  pageNum: number,
  pageSize: number,
  // sortBy: string, ////
  // sortDescending: boolean, ////
) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParams(pageNum, pageSize, null, null, false)

  return axios
    .get<SessionAuditResponse>(`${API_URL}/${SESSIONS_PATH}/${sessionId}/audit?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data) //// uncomment
  // .then((res) => {
  //   //// remove mock
  //   res.data.data.entries.forEach((entry) => {
  //     // mock big json
  //     const bigJson = {
  //       example: 'data',
  //       nested: {
  //         array: [1, 2, 3],
  //         object: { key: 'value' },
  //       },
  //       anotherKey: 'anotherValue',
  //       numbers: [10, 20, 30, 40, 50],
  //       boolean: true,
  //       nullValue: null,
  //       moreData: {
  //         level1: {
  //           level2: {
  //             level3: 'deepValue',
  //           },
  //         },
  //       },
  //       items: Array.from({ length: 20 }, (_, i) => `item${i + 1}`), // creates an array with 20 items
  //     }
  //     entry.request_data = JSON.stringify(bigJson, null, 2) ////
  //   })
  //   // entry.request_data =  ////
  //   return res.data.data
  // }) //// delete
}
