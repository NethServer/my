//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'

export const ALERT_FILTERS_KEY = 'alertFilters'

export interface AlertFilterAlert {
  name: string
  severity: string
  service?: string
}

export interface AlertFiltersData {
  alerts: AlertFilterAlert[]
}

interface AlertFiltersResponse {
  code: number
  message: string
  data: AlertFiltersData
}

export const getAlertFilters = () => {
  const loginStore = useLoginStore()

  return axios
    .get<AlertFiltersResponse>(`${API_URL}/filters/alerts`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
