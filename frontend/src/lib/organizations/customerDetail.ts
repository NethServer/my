//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { Customer } from './customers'

interface CustomerDetailResponse {
  code: number
  message: string
  data: Customer
}

export interface CustomerStats {
  users_count: number
  systems_count: number
  applications_count: number
}

interface CustomerStatsResponse {
  code: number
  message: string
  data: CustomerStats
}

export const getCustomerDetail = (customerId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<CustomerDetailResponse>(`${API_URL}/customers/${customerId}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getCustomerStats = (customerId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<CustomerStatsResponse>(`${API_URL}/customers/${customerId}/stats`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
