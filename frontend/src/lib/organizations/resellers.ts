//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { type Pagination } from '../common'

export const RESELLERS_KEY = 'resellers'
export const RESELLERS_TOTAL_KEY = 'resellersTotal'
export const RESELLERS_TABLE_ID = 'resellersTable'

export type ResellerStatus = 'enabled' | 'suspended' | 'deleted'

export const CreateResellerSchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('organizations.name_cannot_be_empty')),
  custom_data: v.object({
    vat: v.pipe(v.string(), v.nonEmpty('organizations.custom_data_vat_cannot_be_empty')),
    address: v.optional(v.string()),
    city: v.optional(v.string()),
    main_contact: v.optional(v.string()),
    email: v.optional(
      v.union([
        v.literal(''),
        v.pipe(v.string(), v.email('organizations.custom_data_email_invalid')),
      ]),
    ),
    phone: v.optional(
      v.union([
        v.literal(''),
        v.pipe(
          v.string(),
          v.regex(/^\+?[\d\s\-\(\)]{7,20}$/, 'organizations.custom_data_phone_invalid_format'),
        ),
      ]),
    ),
    language: v.optional(v.string()),
    notes: v.optional(v.string()),
  }),
})

export const ResellerSchema = v.object({
  ...CreateResellerSchema.entries,
  logto_id: v.string(),
  suspended_at: v.optional(v.string()),
  deleted_at: v.optional(v.string()),
})

export type CreateReseller = v.InferOutput<typeof CreateResellerSchema>
export type Reseller = v.InferOutput<typeof ResellerSchema>

interface ResellersResponse {
  code: number
  message: string
  data: {
    resellers: Reseller[]
    pagination: Pagination
  }
}

export const getQueryStringParams = (
  pageNum: number,
  pageSize: number,
  textFilter: string | null,
  statusFilter: ResellerStatus[],
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

  statusFilter.forEach((status) => {
    searchParams.append('status', status)
  })

  return searchParams.toString()
}

export const getResellers = (
  pageNum: number,
  pageSize: number,
  textFilter: string,
  statusFilter: ResellerStatus[],
  sortBy: string,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParams(
    pageNum,
    pageSize,
    textFilter,
    statusFilter,
    sortBy,
    sortDescending,
  )

  return axios
    .get<ResellersResponse>(`${API_URL}/resellers?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const postReseller = (reseller: CreateReseller) => {
  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/resellers`, reseller, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const putReseller = (reseller: Reseller) => {
  const loginStore = useLoginStore()

  return axios.put(`${API_URL}/resellers/${reseller.logto_id}`, reseller, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteReseller = (reseller: Reseller) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/resellers/${reseller.logto_id}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const getResellersTotal = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/resellers/totals`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.total as number)
}

export const suspendReseller = (reseller: Reseller) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/resellers/${reseller.logto_id}/suspend`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const reactivateReseller = (reseller: Reseller) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/resellers/${reseller.logto_id}/reactivate`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const restoreReseller = (reseller: Reseller) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/resellers/${reseller.logto_id}/restore`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const getQueryStringParamsForExport = (
  format: string,
  textFilter: string | undefined,
  statusFilter: ResellerStatus[] | undefined,
  sortBy: string | undefined,
  sortDescending: boolean | undefined,
) => {
  const searchParams = new URLSearchParams({
    format: format,
  })

  if (textFilter?.trim()) {
    searchParams.append('search', textFilter)
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
  statusFilter: ResellerStatus[] | undefined = undefined,
  sortBy: string | undefined = undefined,
  sortDescending: boolean | undefined = undefined,
) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParamsForExport(
    format,
    textFilter,
    statusFilter,
    sortBy,
    sortDescending,
  )

  return axios
    .get(`${API_URL}/resellers/export?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data)
}
