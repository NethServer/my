//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const SYSTEM_ORGANIZATION_FILTER_KEY = 'systemOrganizationFilter'

const SYSTEM_ORGANIZATION_FILTER_PATH = 'filters/systems/organizations'

interface OrganizationFilterResponse {
  code: number
  message: string
  data: {
    organizations: OrganizationItem[]
  }
}

interface OrganizationItem {
  id: string
  name: string
}

export const getOrganizationFilter = () => {
  const loginStore = useLoginStore()

  return axios
    .get<OrganizationFilterResponse>(`${API_URL}/${SYSTEM_ORGANIZATION_FILTER_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
