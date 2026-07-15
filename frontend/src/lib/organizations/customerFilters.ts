//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const CUSTOMER_FILTERS_KEY = 'customerFilters'

const CUSTOMER_FILTERS_PATH = 'filters/customers'

export interface CreatedByItem {
  user_id: string
  name: string
  email: string
  organization_name: string
}

export interface CustomerFiltersData {
  created_by: CreatedByItem[]
}

interface CustomerFiltersResponse {
  code: number
  message: string
  data: CustomerFiltersData
}

export const getCustomerFilters = () => {
  const loginStore = useLoginStore()

  return axios
    .get<CustomerFiltersResponse>(`${API_URL}/${CUSTOMER_FILTERS_PATH}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
