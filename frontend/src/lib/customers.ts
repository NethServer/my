//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { getQueryStringParams, type Pagination } from './common'

export const CUSTOMERS_KEY = 'customers'
export const CUSTOMERS_TOTAL_KEY = 'customersTotal'
export const CUSTOMERS_TABLE_ID = 'customersTable'

export const CreateCustomerSchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('organizations.name_cannot_be_empty')),
  description: v.optional(v.string()),
  custom_data: v.object({
    vat: v.pipe(v.string(), v.nonEmpty('organizations.custom_data_vat_cannot_be_empty')),
  }),
})

export const CustomerSchema = v.object({
  ...CreateCustomerSchema.entries,
  logto_id: v.string(),
})

export type CreateCustomer = v.InferOutput<typeof CreateCustomerSchema>
export type Customer = v.InferOutput<typeof CustomerSchema>

interface CustomersResponse {
  code: number
  message: string
  data: {
    customers: Customer[]
    pagination: Pagination
  }
}

export const getCustomers = (
  pageNum: number,
  pageSize: number,
  textFilter: string,
  sortBy: string,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()
  const params = getQueryStringParams(pageNum, pageSize, textFilter, sortBy, sortDescending)

  return axios
    .get<CustomersResponse>(`${API_URL}/customers?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const postCustomer = (customer: CreateCustomer) => {
  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/customers`, customer, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const putCustomer = (customer: Customer) => {
  const loginStore = useLoginStore()

  return axios.put(`${API_URL}/customers/${customer.logto_id}`, customer, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteCustomer = (customer: Customer) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/customers/${customer.logto_id}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const getCustomersTotal = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/customers/totals`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.total as number)
}
