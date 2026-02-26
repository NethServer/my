//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { type Pagination } from '../common'

export const CUSTOMERS_KEY = 'customers'
export const CUSTOMERS_TOTAL_KEY = 'customersTotal'
export const CUSTOMERS_TABLE_ID = 'customersTable'

export type CustomerStatus = 'enabled' | 'suspended' | 'deleted'

export const CreateCustomerSchema = v.object({
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

export const EditCustomerSchema = v.object({
  ...CreateCustomerSchema.entries,
  logto_id: v.string(),
})

export const CustomerSchema = v.object({
  ...CreateCustomerSchema.entries,
  ...EditCustomerSchema.entries,
  suspended_at: v.optional(v.string()),
  deleted_at: v.optional(v.string()),
  systems_count: v.number(),
  customers_count: v.number(),
})

export type CreateCustomer = v.InferOutput<typeof CreateCustomerSchema>
export type EditCustomer = v.InferOutput<typeof EditCustomerSchema>
export type Customer = v.InferOutput<typeof CustomerSchema>

interface CustomersResponse {
  code: number
  message: string
  data: {
    customers: Customer[]
    pagination: Pagination
  }
}

export const getQueryStringParams = (
  pageNum: number,
  pageSize: number,
  textFilter: string | null,
  statusFilter: CustomerStatus[],
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

export const getCustomers = (
  pageNum: number,
  pageSize: number,
  textFilter: string,
  statusFilter: CustomerStatus[],
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

export const putCustomer = (customer: EditCustomer) => {
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

export const destroyCustomer = (customer: Customer) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/customers/${customer.logto_id}/destroy`, {
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

export const suspendCustomer = (customer: Customer) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/customers/${customer.logto_id}/suspend`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const reactivateCustomer = (customer: Customer) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/customers/${customer.logto_id}/reactivate`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const restoreCustomer = (customer: Customer) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/customers/${customer.logto_id}/restore`,
    {},
    {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    },
  )
}

export const getQueryStringParamsForExport = (
  format: string,
  textFilter: string | undefined,
  statusFilter: CustomerStatus[] | undefined,
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
  statusFilter: CustomerStatus[] | undefined = undefined,
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
    .get(`${API_URL}/customers/export?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data)
}
