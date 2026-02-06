//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { getQueryStringParams, type Pagination } from './common'

export const DISTRIBUTORS_KEY = 'distributors'
export const DISTRIBUTORS_TOTAL_KEY = 'distributorsTotal'
export const DISTRIBUTORS_TABLE_ID = 'distributorsTable'

export const CreateDistributorSchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('organizations.name_cannot_be_empty')),
  description: v.optional(v.string()),
  custom_data: v.object({
    vat: v.pipe(v.string(), v.nonEmpty('organizations.custom_data_vat_cannot_be_empty')),
    notes: v.optional(v.string()),
  }),
})

export const DistributorSchema = v.object({
  ...CreateDistributorSchema.entries,
  logto_id: v.string(),
  suspended_at: v.optional(v.string()),
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

export const getDistributors = (
  pageNum: number,
  pageSize: number,
  textFilter: string,
  sortBy: string,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParams(pageNum, pageSize, textFilter, sortBy, sortDescending)

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
