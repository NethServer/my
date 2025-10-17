//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { type Pagination } from '../common'

export const SYSTEMS_KEY = 'systems'
export const SYSTEMS_TOTAL_KEY = 'systemsTotal' //// needed?
export const SYSTEMS_TABLE_ID = 'systemsTable'

export type SystemStatus = 'online' | 'offline' | 'undefined' | 'deleted'

const systemStatusOptions = ['online', 'offline', 'undefined', 'deleted']

const SystemStatusSchema = v.picklist(systemStatusOptions)

export const CreateSystemSchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('systems.name_cannot_be_empty')),
  organization_id: v.pipe(v.string(), v.nonEmpty('systems.organization_required')),
  notes: v.pipe(v.string()),
  custom_data: v.optional(v.record(v.string(), v.string())), //// use correct types
})

export const EditSystemSchema = v.object({
  ...CreateSystemSchema.entries,
  id: v.string(),
})

export const SystemSchema = v.object({
  ...CreateSystemSchema.entries,
  ...EditSystemSchema.entries,
  type: v.string(),
  status: v.optional(SystemStatusSchema),
  fqdn: v.string(),
  ipv4_address: v.string(),
  ipv6_address: v.string(),
  version: v.string(),
  organization_name: v.string(),
  created_at: v.string(),
  updated_at: v.string(),
  system_key: v.optional(v.string()),
  system_secret: v.string(),
  created_by: v.object({
    user_id: v.string(),
    name: v.string(),
    email: v.string(),
    organization_id: v.string(),
    organization_name: v.string(),
  }),
})

////
// export const SystemSchema = v.object({
//   ...CreateSystemSchema.entries,
//   ...EditSystemSchema.entries,
//   type: v.optional(v.string()),
//   organization: v.optional(
//     v.object({
//       id: v.string(),
//       logto_id: v.optional(v.string()),
//       name: v.string(),
//     }),
//   ),
//   roles: v.optional(
//     v.array(
//       v.object({
//         id: v.string(),
//         name: v.string(),
//       }),
//     ),
//   ),
// })

export type CreateSystem = v.InferOutput<typeof CreateSystemSchema>
export type EditSystem = v.InferOutput<typeof EditSystemSchema>
export type System = v.InferOutput<typeof SystemSchema>

interface SystemsResponse {
  code: number
  message: string
  data: {
    systems: System[]
    pagination: Pagination
  }
}

interface PostSystemResponse {
  code: number
  message: string
  data: System
}

export const getQueryStringParams = (
  pageNum: number,
  pageSize: number,
  textFilter: string | null,
  productFilter: string[],
  createdByFilter: string[],
  versionFilter: string[],
  statusFilter: SystemStatus[],
  sortBy: string | null,
  sortDescending: boolean,
) => {
  const searchParams = new URLSearchParams({
    page: pageNum.toString(),
    page_size: pageSize.toString(),
    sort_by: sortBy || '',
    sort_direction: sortDescending ? 'desc' : 'asc',
  })

  if (textFilter?.trim()) {
    searchParams.append('search', textFilter)
  }

  productFilter.forEach((product) => {
    searchParams.append('type', product)
  })

  createdByFilter.forEach((userId) => {
    searchParams.append('created_by', userId)
  })

  versionFilter.forEach((version) => {
    searchParams.append('version', version)
  })

  statusFilter.forEach((status) => {
    searchParams.append('status', status)
  })
  return searchParams.toString()
}

export const getSystems = (
  pageNum: number,
  pageSize: number,
  textFilter: string,
  productFilter: string[],
  createdByFilter: string[],
  versionFilter: string[],
  statusFilter: SystemStatus[],
  sortBy: string,
  sortDescending: boolean,
) => {
  // convert productFilter to string using openapi format
  const loginStore = useLoginStore()
  const params = getQueryStringParams(
    pageNum,
    pageSize,
    textFilter,
    productFilter,
    createdByFilter,
    versionFilter,
    statusFilter,
    sortBy,
    sortDescending,
  )

  return axios
    .get<SystemsResponse>(`${API_URL}/systems?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const postSystem = (system: CreateSystem) => {
  const loginStore = useLoginStore()

  return axios
    .post<PostSystemResponse>(`${API_URL}/systems`, system, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data)
}

export const putSystem = (system: EditSystem) => {
  const loginStore = useLoginStore()

  return axios.put<PostSystemResponse>(`${API_URL}/systems/${system.id}`, system, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteSystem = (system: System) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/systems/${system.id}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

//// used?
export const regenerateSystemSecret = (systemId: string) => {
  const loginStore = useLoginStore()

  return axios.post<System>(
    `${API_URL}/systems/${systemId}/regenerate-secret`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const getSystemsTotal = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/systems/totals`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.total as number)
}
