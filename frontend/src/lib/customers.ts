//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'

//// check attributes
export const CustomerSchema = v.object({
  id: v.optional(v.string()),
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
  customData: v.optional(
    v.object({
      address: v.optional(v.string()),
      city: v.optional(v.string()),
      codiceFiscale: v.optional(v.string()),
      contactPerson: v.optional(v.string()),
      createdAt: v.optional(v.string()),
      createdBy: v.optional(v.string()),
      email: v.optional(v.string()),
      partitaIva: v.optional(v.string()),
      phone: v.optional(v.string()),
      region: v.optional(v.string()),
      territory: v.optional(v.array(v.string())),
      // type: v.literal('customer'), ////
      website: v.optional(v.string()),
    }),
  ),
  // type: v.literal('customer'), ////
  isMfaRequired: v.optional(v.boolean()),
})

export type Customer = v.InferOutput<typeof CustomerSchema>

export const getCustomers = () => {
  console.log('getCustomers') ////

  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/customers`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.customers as Customer[])
}

export const postCustomer = (customer: Customer) => {
  console.log('postCustomer', customer) ////

  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/customers`, customer, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const putCustomer = (customer: Customer) => {
  console.log('putCustomer', customer) ////

  const loginStore = useLoginStore()

  return axios.put(`${API_URL}/customers/${customer.id}`, customer, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteCustomer = (customer: Customer) => {
  console.log('deleteCustomer', customer) ////

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
  found = [
    'address',
    'city',
    'codiceFiscale',
    'contactPerson',
    'email',
    'partitaIva',
    'phone',
    'region',
  ].some((attrName) => {
    const attrValue = customer.customData?.[
      attrName as keyof NonNullable<Customer['customData']>
    ] as string
    return new RegExp(searchString, 'i').test(attrValue?.replace(regex, ''))
  })

  if (found) {
    return true
  } else {
    return false
  }
}
