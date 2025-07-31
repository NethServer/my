//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { paginationQueryString } from './users'

export const CUSTOMERS_KEY = 'customers'
export const CUSTOMERS_TOTAL_KEY = 'customersTotal'

export const CreateCustomerSchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('organizations.name_cannot_be_empty')),
  description: v.optional(v.string()),
  custom_data: v.object({
    vat: v.pipe(
      v.string(),
      v.nonEmpty('organizations.vat_required'),
      v.regex(/^\d{11}$/, 'organizations.vat_invalid'),
    ),
  }),
})

export const CustomerSchema = v.object({
  ...CreateCustomerSchema.entries,
  id: v.string(),
})

export type CreateCustomer = v.InferOutput<typeof CreateCustomerSchema>
export type Customer = v.InferOutput<typeof CustomerSchema>

export const getCustomers = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/customers${paginationQueryString}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.customers as Customer[])
}

export const postCustomer = (customer: CreateCustomer) => {
  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/customers`, customer, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const putCustomer = (customer: Customer) => {
  const loginStore = useLoginStore()

  return axios.put(`${API_URL}/customers/${customer.id}`, customer, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteCustomer = (customer: Customer) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/customers/${customer.id}`, {
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

export const searchStringInCustomer = (
  searchString: string,

  customer: Customer,
): boolean => {
  const regex = /[^a-zA-Z0-9-]/g
  searchString = searchString.replace(regex, '')
  let found = false

  // search in string attributes
  found = ['name', 'description'].some((attrName) => {
    const attrValue = customer[attrName as keyof Customer] as string
    return new RegExp(searchString, 'i').test(attrValue?.replace(regex, ''))
  })

  if (found) {
    return true
  }

  //// review customData attributes

  // search in customData
  found = ['address', 'city', 'codiceFiscale', 'email', 'partitaIva', 'phone', 'region'].some(
    (attrName) => {
      const attrValue = customer.custom_data?.[
        attrName as keyof NonNullable<Customer['custom_data']>
      ] as string
      return new RegExp(searchString, 'i').test(attrValue?.replace(regex, ''))
    },
  )

  if (found) {
    return true
  } else {
    return false
  }
}
