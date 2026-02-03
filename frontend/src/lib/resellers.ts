//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { getQueryStringParams, type Pagination } from './common'

export const RESELLERS_KEY = 'resellers'
export const RESELLERS_TOTAL_KEY = 'resellersTotal'
export const RESELLERS_TABLE_ID = 'resellersTable'

export const CreateResellerSchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('organizations.name_cannot_be_empty')),
  description: v.optional(v.string()),
  custom_data: v.object({
    vat: v.pipe(v.string(), v.nonEmpty('organizations.custom_data_vat_cannot_be_empty')),
  }),
})

export const ResellerSchema = v.object({
  ...CreateResellerSchema.entries,
  logto_id: v.string(),
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

export const getResellers = (
  pageNum: number,
  pageSize: number,
  textFilter: string,
  sortBy: string,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParams(pageNum, pageSize, textFilter, sortBy, sortDescending)

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
