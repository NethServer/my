//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'

//// check attributes
export const CreateDistributorSchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('distributors.name_required')),
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

export const DistributorSchema = v.object({
  ...CreateDistributorSchema.entries,
  id: v.string(),
})

export type CreateDistributor = v.InferOutput<typeof CreateDistributorSchema>
export type Distributor = v.InferOutput<typeof DistributorSchema>

export const getDistributors = () => {
  console.log('getDistributors') ////

  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/distributors`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.distributors as Distributor[])
}

export const postDistributor = (distributor: CreateDistributor) => {
  console.log('postDistributor', distributor) ////

  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/distributors`, distributor, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const putDistributor = (distributor: Distributor) => {
  console.log('putDistributor', distributor) ////

  const loginStore = useLoginStore()

  return axios.put(`${API_URL}/distributors/${distributor.id}`, distributor, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteDistributor = (distributor: Distributor) => {
  console.log('deleteDistributor', distributor) ////

  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/distributors/${distributor.id}`, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const searchStringInDistributor = (
  searchString: string,

  distributor: Distributor,
): boolean => {
  const regex = /[^a-zA-Z0-9-]/g
  searchString = searchString.replace(regex, '')
  let found = false

  // search in string attributes
  found = ['name', 'description'].some((attrName) => {
    const attrValue = distributor[attrName as keyof Distributor] as string
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
    const attrValue = distributor.customData?.[
      attrName as keyof NonNullable<Distributor['customData']>
    ] as string
    return new RegExp(searchString, 'i').test(attrValue?.replace(regex, ''))
  })

  if (found) {
    return true
  } else {
    return false
  }
}
