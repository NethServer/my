//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from './config'
import { useLoginStore } from '@/stores/login'

export const ALERT_FILTERS_KEY = 'alertFilters'

export interface AlertFilterSystem {
  id: string
  name: string
  type: string
  key: string
}

export interface AlertFilterAlert {
  name: string
  severity: string
  service?: string
}

export interface AlertFilterOrganization {
  logto_id: string
  name: string
  type: string
}

export interface AlertFiltersData {
  systems: AlertFilterSystem[]
  alerts: AlertFilterAlert[]
  severities: string[]
  organizations: AlertFilterOrganization[]
}

interface AlertFiltersResponse {
  code: number
  message: string
  data: AlertFiltersData
}

export const getAlertFilters = (
  organizationIds?: string[],
  include: 'descendants' = 'descendants',
) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams()

  if (organizationIds?.length) {
    organizationIds.forEach((id) => params.append('organization_id', id))
  }

  if (include === 'descendants') {
    params.append('include', 'descendants')
  }

  return axios
    .get<AlertFiltersResponse>(`${API_URL}/filters/alerts?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
