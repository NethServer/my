//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'
import { paginationQueryString } from './users'

export const DISTRIBUTORS_KEY = 'distributors'
export const DISTRIBUTORS_TOTAL_KEY = 'distributorsTotal'

export const CreateDistributorSchema = v.object({
  name: v.pipe(v.string(), v.nonEmpty('organizations.name_cannot_be_empty')),
  description: v.optional(v.string()),
  custom_data: v.object({
    vat: v.pipe(
      v.string(),
      v.nonEmpty('organizations.custom_data_vat_cannot_be_empty'),
      v.regex(/^\d{11}$/, 'organizations.custom_data_vat_invalid'),
    ),
  }),
})

export const DistributorSchema = v.object({
  ...CreateDistributorSchema.entries,
  id: v.string(),
})

export type CreateDistributor = v.InferOutput<typeof CreateDistributorSchema>
export type Distributor = v.InferOutput<typeof DistributorSchema>

export const getDistributors = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/distributors${paginationQueryString}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.distributors as Distributor[])
}

export const postDistributor = (distributor: CreateDistributor) => {
  const loginStore = useLoginStore()

  return axios.post(`${API_URL}/distributors`, distributor, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const putDistributor = (distributor: Distributor) => {
  const loginStore = useLoginStore()

  return axios.put(`${API_URL}/distributors/${distributor.id}`, distributor, {
    headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
  })
}

export const deleteDistributor = (distributor: Distributor) => {
  const loginStore = useLoginStore()

  return axios.delete(`${API_URL}/distributors/${distributor.id}`, {
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
  found = ['address', 'city', 'codiceFiscale', 'email', 'partitaIva', 'phone', 'region'].some(
    (attrName) => {
      const attrValue = distributor.custom_data?.[
        attrName as keyof NonNullable<Distributor['custom_data']>
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
