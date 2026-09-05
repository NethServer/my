//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import { OPTIONS_PAGE_SIZE, type Pagination } from '../common'

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

/**
 * @param types restricts the results to these company types (distributor,
 *   reseller, customer); the values are OR-ed. Scoping server-side keeps the
 *   paginated option list meaningful: filtering by type on the client would drop
 *   entries from an already-truncated page.
 */
export const getOrganizationsSearchQueryString = (search: string, types?: string[]): string => {
  const params = new URLSearchParams({
    page: '1',
    page_size: OPTIONS_PAGE_SIZE.toString(),
  })

  if (search.trim()) {
    params.append('search', search)
  }

  // `type` is a repeated parameter on the wire — the backend reads it with
  // c.QueryArray and does not split on commas, so joining the values would send
  // one literal "a,b" filter that matches nothing.
  for (const type of types ?? []) {
    params.append('type', type)
  }

  return params.toString()
}

export const searchOrganizations = (search: string, types?: string[]) => {
  const loginStore = useLoginStore()
  const params = getOrganizationsSearchQueryString(search, types)

  return axios
    .get<OrganizationsSearchResponse>(`${API_URL}/organizations?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data.organizations)
}
