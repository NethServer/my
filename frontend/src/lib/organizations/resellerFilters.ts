//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const RESELLER_FILTERS_KEY = 'resellerFilters'

const RESELLER_FILTERS_PATH = 'filters/resellers'

export interface CreatedByItem {
  user_id: string
  name: string
  email: string
  organization_name: string
}

export interface ResellerFiltersData {
  created_by: CreatedByItem[]
}

interface ResellerFiltersResponse {
  code: number
  message: string
  data: ResellerFiltersData
}

export const getResellerFilters = () => {
  const loginStore = useLoginStore()

  return axios
    .get<ResellerFiltersResponse>(`${API_URL}/${RESELLER_FILTERS_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
