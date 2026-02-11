//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { type Pagination } from '../common'

export const USERS_KEY = 'users'
export const USERS_TOTAL_KEY = 'usersTotal'
export const USERS_TABLE_ID = 'usersTable'

export type UserStatus = 'enabled' | 'suspended' | 'deleted'

export const CreateUserSchema = v.object({
  email: v.pipe(v.string(), v.nonEmpty('users.email_required'), v.email('users.email_invalid')),
  name: v.pipe(v.string(), v.nonEmpty('users.name_cannot_be_empty')),
  phone: v.optional(
    v.union([
      v.literal(''),
      v.pipe(v.string(), v.regex(/^\+?[\d\s\-\(\)]{7,20}$/, 'users.phone_invalid_format')),
    ]),
  ),
  user_role_ids: v.optional(v.array(v.string())),
  organization_id: v.pipe(v.string(), v.nonEmpty('users.organization_required')),
  custom_data: v.optional(v.record(v.string(), v.string())), //// use correct types
})

export const EditUserSchema = v.object({
  ...CreateUserSchema.entries,
  logto_id: v.string(),
})

export const UserSchema = v.object({
  ...CreateUserSchema.entries,
  ...EditUserSchema.entries,
  active: v.optional(v.boolean()),
  logto_id: v.optional(v.string()),
  can_be_impersonated: v.boolean(),
  logto_synced_at: v.optional(v.string()),
  suspended_at: v.optional(v.string()),
  deleted_at: v.optional(v.string()),
  organization: v.object({
    id: v.string(),
    logto_id: v.optional(v.string()),
    name: v.string(),
    type: v.string(),
  }),
  roles: v.optional(
    v.array(
      v.object({
        id: v.string(),
        name: v.string(),
      }),
    ),
  ),
})

export type CreateUser = v.InferOutput<typeof CreateUserSchema>
export type EditUser = v.InferOutput<typeof EditUserSchema>
export type User = v.InferOutput<typeof UserSchema>

interface UsersResponse {
  code: number
  message: string
  data: {
    users: User[]
    pagination: Pagination
  }
}

export const getQueryStringParams = (
  pageNum: number,
  pageSize: number,
  textFilter: string | null,
  organizationFilter: string[],
  statusFilter: UserStatus[],
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

  organizationFilter.forEach((orgId) => {
    searchParams.append('organization_id', orgId)
  })

  statusFilter.forEach((status) => {
    searchParams.append('status', status)
  })
  return searchParams.toString()
}

export const getUsers = (
  pageNum: number,
  pageSize: number,
  textFilter: string,
  organizationFilter: string[],
  statusFilter: UserStatus[],
  sortBy: string,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParams(
    pageNum,
    pageSize,
    textFilter,
    organizationFilter,
    statusFilter,
    sortBy,
    sortDescending,
  )

  return axios
    .get<UsersResponse>(`${API_URL}/users?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const postUser = (user: CreateUser) => {
  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/users`, user, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const putUser = (user: EditUser) => {
  const loginStore = useLoginStore()

  return axios.put(`${API_URL}/users/${user.logto_id}`, user, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteUser = (user: User) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/users/${user.logto_id}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

//// add typing
export const getUsersTotal = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/users/totals`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.total as number)
}

export const resetPassword = (user: User, newPassword: string) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/users/${user.logto_id}/password`,
    { password: newPassword },
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const suspendUser = (user: User) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/users/${user.logto_id}/suspend`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const reactivateUser = (user: User) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/users/${user.logto_id}/reactivate`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

//// TODO wait for backend fix
// export const getExport = (
//   format: 'csv' | 'pdf',
//   textFilter: string | undefined = undefined,
//   roleFilter: string[] | undefined = undefined,
//   organizationFilter: string[] | undefined = undefined,
//   statusFilter: SystemStatus[] | undefined = undefined,
//   sortBy: string | undefined = undefined,
//   sortDescending: boolean | undefined = undefined,
// ) => {
//   const loginStore = useLoginStore()
//   const params = getQueryStringParamsForExport(
//     format,
//     textFilter,
//     roleFilter,
//     organizationFilter,
//     statusFilter,
//     sortBy,
//     sortDescending,
//   )

//   return axios
//     .get(`${API_URL}/systems/export?${params}`, {
//       headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
//     })
//     .then((res) => res.data)
// }
