//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { getQueryStringParams, type Pagination } from './common'

export const USERS_KEY = 'users'
export const USERS_TOTAL_KEY = 'usersTotal'
export const USERS_TABLE_ID = 'usersTable'

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
  id: v.string(),
})

export const UserSchema = v.object({
  ...CreateUserSchema.entries,
  ...EditUserSchema.entries,
  active: v.optional(v.boolean()),
  logto_id: v.optional(v.string()),
  can_be_impersonated: v.boolean(),
  logto_synced_at: v.optional(v.string()),
  suspended_at: v.optional(v.string()),
  organization: v.optional(
    v.object({
      id: v.string(),
      logto_id: v.optional(v.string()),
      name: v.string(),
    }),
  ),
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

export const getUsers = (
  pageNum: number,
  pageSize: number,
  textFilter: string,
  sortBy: string,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParams(pageNum, pageSize, textFilter, sortBy, sortDescending)

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

  return axios.put(`${API_URL}/users/${user.id}`, user, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteUser = (user: User) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/users/${user.id}`, {
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
    `${API_URL}/users/${user.id}/password`,
    { password: newPassword },
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const suspendUser = (user: User) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/users/${user.id}/suspend`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const reactivateUser = (user: User) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/users/${user.id}/reactivate`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}
