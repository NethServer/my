//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'

//// check attributes
export const ResellerSchema = v.object({
  id: v.optional(v.string()),
  name: v.pipe(v.string(), v.nonEmpty('resellers.name_required')),
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
      website: v.optional(v.string()),
    }),
  ),
  isMfaRequired: v.optional(v.boolean()),
})

export type Reseller = v.InferOutput<typeof ResellerSchema>

export const getResellers = () => {
  console.log('getResellers') ////

  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/resellers`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.resellers as Reseller[])
}

export const postReseller = (reseller: Reseller) => {
  console.log('postReseller', reseller) ////

  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/resellers`, reseller, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const putReseller = (reseller: Reseller) => {
  console.log('putReseller', reseller) ////

  const loginStore = useLoginStore()

  return axios.put(`${API_URL}/resellers/${reseller.id}`, reseller, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteReseller = (reseller: Reseller) => {
  console.log('deleteReseller', reseller) ////

  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/resellers/${reseller.id}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const searchStringInReseller = (
  searchString: string,

  reseller: Reseller,
): boolean => {
  const regex = /[^a-zA-Z0-9-]/g
  searchString = searchString.replace(regex, '')
  let found = false

  // search in string attributes
  found = ['name', 'description'].some((attrName) => {
    const attrValue = reseller[attrName as keyof Reseller] as string
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
    const attrValue = reseller.customData?.[
      attrName as keyof NonNullable<Reseller['customData']>
    ] as string
    return new RegExp(searchString, 'i').test(attrValue?.replace(regex, ''))
  })

  if (found) {
    return true
  } else {
    return false
  }
}
