//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { paginationQueryString } from './users'

//// check attributes
export const CreateCustomerSchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('customers.name_required')),
  description: v.optional(v.string()),
  branding: v.optional(
    v.object({
      darkFavicon: v.string(),
      darkLogoUrl: v.string(),
      favicon: v.string(),
      logoUrl: v.string(),
    }),
  ),
  customData: v.optional(v.record(v.string(), v.string())),
  isMfaRequired: v.optional(v.boolean()),
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
      const attrValue = customer.customData?.[
        attrName as keyof NonNullable<Customer['customData']>
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
