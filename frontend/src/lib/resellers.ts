//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { paginationQueryString } from './users'

export const CreateResellerSchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('organizations.name_required')),
  description: v.optional(v.string()),
  custom_data: v.object({
    vat_number: v.pipe(
      v.string(),
      v.nonEmpty('organizations.vat_number_required'),
      v.regex(/^\d{11}$/, 'organizations.vat_number_invalid'),
    ),
  }),
})

export const ResellerSchema = v.object({
  ...CreateResellerSchema.entries,
  id: v.string(),
})

export type CreateReseller = v.InferOutput<typeof CreateResellerSchema>
export type Reseller = v.InferOutput<typeof ResellerSchema>

export const getResellers = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/resellers${paginationQueryString}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.resellers as Reseller[])
}

export const postReseller = (reseller: CreateReseller) => {
  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/resellers`, reseller, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const putReseller = (reseller: Reseller) => {
  const loginStore = useLoginStore()

  return axios.put(`${API_URL}/resellers/${reseller.id}`, reseller, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteReseller = (reseller: Reseller) => {
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
  found = ['address', 'city', 'codiceFiscale', 'email', 'partitaIva', 'phone', 'region'].some(
    (attrName) => {
      const attrValue = reseller.custom_data?.[
        attrName as keyof NonNullable<Reseller['custom_data']>
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
