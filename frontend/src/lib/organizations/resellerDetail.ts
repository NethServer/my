//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { Reseller } from './resellers'

interface ResellerDetailResponse {
  code: number
  message: string
  data: Reseller
}

export interface ResellerStats {
  users_count: number
  systems_count: number
  customers_count: number
  applications_count: number
  applications_hierarchy_count: number
  systems_hierarchy_count: number
}

interface ResellerStatsResponse {
  code: number
  message: string
  data: ResellerStats
}

export const getResellerDetail = (resellerId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<ResellerDetailResponse>(`${API_URL}/resellers/${resellerId}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getResellerStats = (resellerId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<ResellerStatsResponse>(`${API_URL}/resellers/${resellerId}/stats`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
