//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export interface DistributorDetail {
  id: string
  logto_id: string
  name: string
  description: string
  custom_data: {
    address: string
    city: string
    createdAt: string
    createdBy: string
    email: string
    language: string
    main_contact: string
    notes: string
    phone: string
    type: string
    updatedAt: string
    updatedBy: string
    vat: string
  }
  created_at: string
  updated_at: string
  logto_synced_at: string | null
  logto_sync_error: string | null
  deleted_at: string | null
  suspended_at: string | null
  rebranding_enabled: boolean
}

interface DistributorDetailResponse {
  code: number
  message: string
  data: DistributorDetail
}

export interface DistributorStats {
  users_count: number
  systems_count: number
  resellers_count: number
  customers_count: number
  applications_count: number
  applications_hierarchy_count: number
  systems_hierarchy_count: number
}

interface DistributorStatsResponse {
  code: number
  message: string
  data: DistributorStats
}

export const getDistributorDetail = (distributorId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<DistributorDetailResponse>(`${API_URL}/distributors/${distributorId}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getDistributorStats = (distributorId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<DistributorStatsResponse>(`${API_URL}/distributors/${distributorId}/stats`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
