//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const APPLICATION_ORGANIZATION_FILTER_KEY = 'applicationOrganizationFilter'

const APPLICATION_ORGANIZATION_FILTER_PATH = 'filters/applications/organizations'

interface OrganizationFilterResponse {
  code: number
  message: string
  data: ApplicationOrganization[]
}

interface ApplicationOrganization {
  id: string
  logto_id: string
  name: string
  description: string
  type: string
}

export const getOrganizationFilter = () => {
  const loginStore = useLoginStore()

  return axios
    .get<OrganizationFilterResponse>(`${API_URL}/${APPLICATION_ORGANIZATION_FILTER_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
