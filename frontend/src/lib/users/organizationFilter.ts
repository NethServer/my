//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const USERS_ORGANIZATION_FILTER_KEY = 'usersOrganizationFilter'

const USERS_ORGANIZATION_FILTER_PATH = 'filters/users/organizations'

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
    .get<OrganizationFilterResponse>(`${API_URL}/${USERS_ORGANIZATION_FILTER_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
