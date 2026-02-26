//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { UserRole } from './userRoles'

export const USER_FILTERS_KEY = 'userFilters'

const USER_FILTERS_PATH = 'filters/users'

export interface OrganizationItem {
  id: string
  name: string
  type: string
}

export interface UserFiltersData {
  roles: UserRole[]
  organizations: OrganizationItem[]
}

interface UserFiltersResponse {
  code: number
  message: string
  data: UserFiltersData
}

export const getUserFilters = () => {
  const loginStore = useLoginStore()

  return axios
    .get<UserFiltersResponse>(`${API_URL}/${USER_FILTERS_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
