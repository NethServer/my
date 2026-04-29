//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { downloadFile, type Pagination } from '../common'
import { parsePhoneNumberFromString } from 'libphonenumber-js'

export const USERS_KEY = 'users'
export const USERS_TOTAL_KEY = 'usersTotal'
export const USERS_TABLE_ID = 'usersTable'

export type UserStatus = 'enabled' | 'suspended' | 'deleted'

export const PhoneNumberSchema = v.pipe(
  v.string(),
  v.regex(/^\+?[\d\s\-\(\)]{7,20}$/, 'users.phone_invalid_format'),
)

export const CreateUserSchema = v.object({
  email: v.pipe(v.string(), v.nonEmpty('users.email_required'), v.email('users.email_invalid')),
  name: v.pipe(v.string(), v.nonEmpty('users.name_cannot_be_empty')),
  phone: v.optional(v.union([v.literal(''), PhoneNumberSchema])),
  user_role_ids: v.pipe(
    v.array(v.string()),
    v.minLength(1, 'users.user_role_ids_at_least_one_role_is_required'),
  ),
  organization_id: v.pipe(v.string(), v.nonEmpty('users.organization_required')),
  custom_data: v.optional(v.record(v.string(), v.string())),
})

export const EditUserSchema = v.object({
  ...CreateUserSchema.entries,
  logto_id: v.string(),
})

export const UserSchema = v.object({
  ...CreateUserSchema.entries,
  ...EditUserSchema.entries,
  logto_id: v.optional(v.string()),
  username: v.string(),
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

interface UsersTotalResponse {
  code: number
  message: string
  data: {
    total: number
  }
}

export const getQueryStringParams = (
  pageNum: number,
  pageSize: number,
  textFilter: string | null,
  organizationFilter: string[],
  roleFilter: string[],
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

  roleFilter.forEach((roleId) => {
    searchParams.append('role', roleId)
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
  roleFilter: string[],
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
    roleFilter,
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

export const destroyUser = (user: User) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/users/${user.logto_id}/destroy`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const getUsersTotal = () => {
  const loginStore = useLoginStore()

  return axios
    .get<UsersTotalResponse>(`${API_URL}/users/totals`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.total)
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

export const restoreUser = (user: User) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/users/${user.logto_id}/restore`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const getQueryStringParamsForExport = (
  format: string,
  textFilter: string | undefined,
  organizationFilter: string[] | undefined,
  roleFilter: string[] | undefined,
  statusFilter: UserStatus[] | undefined,
  sortBy: string | undefined,
  sortDescending: boolean | undefined,
) => {
  const searchParams = new URLSearchParams({
    format: format,
  })

  if (textFilter?.trim()) {
    searchParams.append('search', textFilter)
  }

  if (organizationFilter) {
    organizationFilter.forEach((orgId) => {
      searchParams.append('organization_id', orgId)
    })
  }

  if (roleFilter) {
    roleFilter.forEach((roleId) => {
      searchParams.append('role', roleId)
    })
  }

  if (statusFilter) {
    statusFilter.forEach((status) => {
      searchParams.append('status', status)
    })
  }

  if (sortBy) {
    searchParams.append('sort_by', sortBy)
  }

  if (sortDescending !== undefined) {
    searchParams.append('sort_direction', sortDescending ? 'desc' : 'asc')
  }

  return searchParams.toString()
}

export const getExport = (
  format: 'csv' | 'pdf',
  textFilter: string | undefined = undefined,
  organizationFilter: string[] | undefined = undefined,
  roleFilter: string[] | undefined = undefined,
  statusFilter: UserStatus[] | undefined = undefined,
  sortBy: string | undefined = undefined,
  sortDescending: boolean | undefined = undefined,
) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParamsForExport(
    format,
    textFilter,
    organizationFilter,
    roleFilter,
    statusFilter,
    sortBy,
    sortDescending,
  )

  return axios
    .get(`${API_URL}/users/export?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data)
}

export async function exportUser(user: User, format: 'pdf' | 'csv') {
  try {
    const exportData = await getExport(format, user.email)
    const fileName = `${user.name}.${format}`
    downloadFile(exportData, fileName, format)
  } catch (error) {
    console.error(`Cannot export user to ${format}:`, error)
    throw error
  }
}

// ============================================================
// Import types
// ============================================================

export interface ImportFieldWarning {
  field: string
  message: string
  value: string
}

export interface ImportFieldError {
  field: string
  message: string
  values: string[]
  candidates?: ImportOrgCandidate[]
}

export interface ImportOrgCandidate {
  logto_id: string
  name: string
  type: string
}

export interface ImportRow {
  row_number: number
  status: 'valid' | 'error' | 'warning' | 'ambiguous'
  data: Record<string, unknown>
  errors?: ImportFieldError[]
  warnings?: ImportFieldWarning[]
}

export interface ImportValidationResult {
  import_id: string
  total_rows: number
  valid_rows: number
  error_rows: number
  warning_rows: number
  ambiguous_rows: number
  rows: ImportRow[]
}

export interface ImportResultRow {
  row_number: number
  status: 'created' | 'updated' | 'skipped' | 'failed'
  id?: string
  reason?: string
  error?: string
}

export interface ImportConfirmResult {
  created: number
  updated: number
  skipped: number
  failed: number
  results: ImportResultRow[]
}

// ============================================================
// Import API functions
// ============================================================

export const getImportTemplate = () => {
  const loginStore = useLoginStore()
  return axios
    .get<Blob>(`${API_URL}/users/import/template`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      responseType: 'blob',
    })
    .then((res) => res.data)
}

export const validateUsersImport = (file: File) => {
  const loginStore = useLoginStore()
  const formData = new FormData()
  formData.append('file', file)
  return axios
    .post<{ code: number; message: string; data: ImportValidationResult }>(
      `${API_URL}/users/import/validate`,
      formData,
      {
        headers: {
          Authorization: `Bearer ${loginStore.jwtToken}`,
          'Content-Type': null,
        },
      },
    )
    .then((res) => res.data.data)
}

export const confirmUsersImport = (
  importId: string,
  override: boolean,
  resolutions?: Record<string, { organization_id: string }>,
) => {
  const loginStore = useLoginStore()
  return axios
    .post<{ code: number; message: string; data: ImportConfirmResult }>(
      `${API_URL}/users/import/confirm`,
      { import_id: importId, override, resolutions },
      {
        headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      },
    )
    .then((res) => res.data.data)
}

// The backend stores phone numbers as digits-only (E.164 without the leading
// `+`), e.g. "393330001113". This helper renders a human-readable version for
// display, e.g. "+39 333 000 1113".
//
// Inputs we need to handle:
// - "" / null / undefined → return ""
// - "393330001113"        → "+39 333 000 1113"
// - "+39 333 0001113"     → "+39 333 000 1113" (already-formatted legacy values
//                           still in the DB get re-parsed and re-formatted)
// - non-parseable values  → returned untouched as a fallback
export function formatPhoneForDisplay(raw: string | null | undefined): string {
  if (!raw) {
    return ''
  }
  // libphonenumber-js needs a leading "+" to detect the country from the digits.
  // If the raw value already has it, parse as-is; otherwise prepend.
  const candidate = raw.startsWith('+') ? raw : `+${raw}`
  const parsed = parsePhoneNumberFromString(candidate)
  return parsed?.formatInternational() ?? raw
}
