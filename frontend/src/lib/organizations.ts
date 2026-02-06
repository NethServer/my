//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'
import { faBuilding, faCity, faCrown, faGlobe, faQuestion } from '@fortawesome/free-solid-svg-icons'
import * as v from 'valibot'

export const ORGANIZATIONS_KEY = 'organizations'

export const OrganizationSchema = v.object({
  logto_id: v.string(),
  name: v.string(),
  description: v.string(),
  type: v.string(),
})

export type Organization = v.InferOutput<typeof OrganizationSchema>

export const getOrganizations = () => {
  const loginStore = useLoginStore()

  return axios
    .get(`${API_URL}/organizations`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.organizations as Organization[])
}

export function getOrganizationIcon(orgType: string) {
  switch (orgType.toLowerCase()) {
    case 'owner':
      return faCrown
    case 'distributor':
      return faGlobe
    case 'reseller':
      return faCity
    case 'customer':
      return faBuilding
    default:
      return faQuestion
  }
}
