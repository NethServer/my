//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const DISTRIBUTOR_FILTERS_KEY = 'distributorFilters'

const DISTRIBUTOR_FILTERS_PATH = 'filters/distributors'

export interface CreatedByItem {
  user_id: string
  name: string
  email: string
  organization_name: string
}

export interface DistributorFiltersData {
  created_by: CreatedByItem[]
}

interface DistributorFiltersResponse {
  code: number
  message: string
  data: DistributorFiltersData
}

export const getDistributorFilters = () => {
  const loginStore = useLoginStore()

  return axios
    .get<DistributorFiltersResponse>(`${API_URL}/${DISTRIBUTOR_FILTERS_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
