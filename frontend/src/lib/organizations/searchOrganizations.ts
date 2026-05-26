//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { Pagination } from '../common'

export const ORGANIZATIONS_SEARCH_KEY = 'organizationsSearch'

export interface OrganizationSearchResult {
  logto_id: string
  name: string
  type: string
}

interface OrganizationsSearchResponse {
  code: number
  message: string
  data: {
    organizations: OrganizationSearchResult[]
    pagination: Pagination
  }
}

export const searchOrganizations = (search: string) => {
  const loginStore = useLoginStore()
  const params = new URLSearchParams({
    page: '1',
    page_size: '50',
  })

  if (search.trim()) {
    params.append('search', search)
  }

  return axios
    .get<OrganizationsSearchResponse>(`${API_URL}/organizations?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.organizations)
}
