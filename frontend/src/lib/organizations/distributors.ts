//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { type Pagination } from '../common'

export const DISTRIBUTORS_KEY = 'distributors'
export const DISTRIBUTORS_TOTAL_KEY = 'distributorsTotal'
export const DISTRIBUTORS_TABLE_ID = 'distributorsTable'

export type DistributorStatus = 'enabled' | 'suspended' | 'deleted'

export const CreateDistributorSchema = v.object({
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

export const DistributorSchema = v.object({
  ...CreateDistributorSchema.entries,
  logto_id: v.string(),
  suspended_at: v.optional(v.string()),
  deleted_at: v.optional(v.string()),
})

export type CreateDistributor = v.InferOutput<typeof CreateDistributorSchema>
export type Distributor = v.InferOutput<typeof DistributorSchema>

interface DistributorsResponse {
  code: number
  message: string
  data: {
    distributors: Distributor[]
    pagination: Pagination
  }
}

export const getQueryStringParams = (
  pageNum: number,
  pageSize: number,
  textFilter: string | null,
  statusFilter: DistributorStatus[],
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

export const getDistributors = (
  pageNum: number,
  pageSize: number,
  textFilter: string,
  statusFilter: DistributorStatus[],
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
    .get<DistributorsResponse>(`${API_URL}/distributors?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const postDistributor = (distributor: CreateDistributor) => {
  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/distributors`, distributor, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const putDistributor = (distributor: Distributor) => {
  const loginStore = useLoginStore()

  return axios.put(`${API_URL}/distributors/${distributor.logto_id}`, distributor, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteDistributor = (distributor: Distributor) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/distributors/${distributor.logto_id}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const getDistributorsTotal = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/distributors/totals`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.total as number)
}

export const suspendDistributor = (distributor: Distributor) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/distributors/${distributor.logto_id}/suspend`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const reactivateDistributor = (distributor: Distributor) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/distributors/${distributor.logto_id}/reactivate`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const restoreDistributor = (distributor: Distributor) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/distributors/${distributor.logto_id}/restore`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const getQueryStringParamsForExport = (
  format: string,
  textFilter: string | undefined,
  statusFilter: DistributorStatus[] | undefined,
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
  statusFilter: DistributorStatus[] | undefined = undefined,
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
    .get(`${API_URL}/distributors/export?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data)
}
