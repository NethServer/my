//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import * as v from 'valibot'
import { API_URL } from '../config'
import { OPTIONS_PAGE_SIZE, type Pagination } from '../common'
import { useLoginStore } from '@/stores/login'

export const REBRANDING_ORGANIZATION_TYPES = ['distributor', 'reseller', 'customer'] as const

export type RebrandingOrganizationType = (typeof REBRANDING_ORGANIZATION_TYPES)[number]

// The only columns the backend knows how to sort on; anything else silently
// falls back to name, which would make a header look broken.
export const REBRANDING_SORT_KEYS = [
  'name',
  'organization_type',
  'enabled_at',
  'updated_at',
] as const

export type RebrandingSortBy = (typeof REBRANDING_SORT_KEYS)[number]

export const RebrandingOrganizationProductSchema = v.object({
  product_id: v.string(),
  product_display_name: v.string(),
  product_name: v.nullable(v.string()),
})

export const RebrandingOrganizationSchema = v.object({
  id: v.string(),
  logto_id: v.string(),
  name: v.string(),
  organization_type: v.picklist(REBRANDING_ORGANIZATION_TYPES),
  products: v.array(RebrandingOrganizationProductSchema),
  enabled_at: v.string(),
  updated_at: v.nullable(v.string()),
})

export const RebrandingSummarySchema = v.object({
  total: v.number(),
  distributors: v.number(),
  resellers: v.number(),
  customers: v.number(),
})

export const RebrandingAvailableOrganizationSchema = v.object({
  id: v.string(),
  logto_id: v.string(),
  name: v.string(),
  organization_type: v.picklist(REBRANDING_ORGANIZATION_TYPES),
})

export type RebrandingOrganizationProduct = v.InferOutput<
  typeof RebrandingOrganizationProductSchema
>
export type RebrandingOrganization = v.InferOutput<typeof RebrandingOrganizationSchema>
export type RebrandingSummary = v.InferOutput<typeof RebrandingSummarySchema>
export type RebrandingAvailableOrganization = v.InferOutput<
  typeof RebrandingAvailableOrganizationSchema
>

interface RebrandingOrganizationsResponse {
  code: number
  message: string
  data: {
    organizations: RebrandingOrganization[]
    pagination: Pagination
  }
}

interface RebrandingSummaryResponse {
  code: number
  message: string
  data: RebrandingSummary
}

interface RebrandingAvailableOrganizationsResponse {
  code: number
  message: string
  data: { organizations: RebrandingAvailableOrganization[] }
}

interface EnableRebrandingResponse {
  code: number
  message: string
  data: { enabled: number }
}

// `type` is a repeated parameter on the wire — the backend reads it with
// c.QueryArray and does not split on commas, so joining the values would send
// one literal "a,b" filter that matches nothing.
export const getRebrandingOrganizationsQueryString = (
  pageNum: number,
  pageSize: number,
  textFilter: string | null,
  typeFilter: string[],
  sortBy: string | null,
  sortDescending: boolean,
): string => {
  const searchParams = new URLSearchParams({
    page: pageNum.toString(),
    page_size: pageSize.toString(),
    sort_by: sortBy || '',
    sort_direction: sortDescending ? 'desc' : 'asc',
  })

  if (textFilter?.trim()) {
    searchParams.append('search', textFilter)
  }

  for (const type of typeFilter) {
    searchParams.append('type', type)
  }

  return searchParams.toString()
}

export const getRebrandingOrganizations = (
  pageNum: number,
  pageSize: number,
  textFilter: string | null,
  typeFilter: string[],
  sortBy: string | null,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()
  const params = getRebrandingOrganizationsQueryString(
    pageNum,
    pageSize,
    textFilter,
    typeFilter,
    sortBy,
    sortDescending,
  )

  return axios
    .get<RebrandingOrganizationsResponse>(`${API_URL}/rebranding/organizations?${params}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

export const getRebrandingSummary = (): Promise<RebrandingSummary> => {
  const loginStore = useLoginStore()

  return axios
    .get<RebrandingSummaryResponse>(`${API_URL}/rebranding/summary`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}

// A picker endpoint, not a list: no pagination, capped server-side at 200.
// Already-enabled organizations and the owner organization are excluded there.
export const getAvailableRebrandingOrganizations = (
  search: string,
  limit: number = OPTIONS_PAGE_SIZE,
): Promise<RebrandingAvailableOrganization[]> => {
  const loginStore = useLoginStore()
  const searchParams = new URLSearchParams({ limit: limit.toString() })

  if (search.trim()) {
    searchParams.append('search', search)
  }

  return axios
    .get<RebrandingAvailableOrganizationsResponse>(
      `${API_URL}/rebranding/organizations/available?${searchParams.toString()}`,
      { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } },
    )
    .then((res) => res.data.data.organizations)
}

// Organization ids are Logto ids: the backend resolves them against
// unified_organizations.logto_id, and sending the numeric id answers
// 400 "organization_ids unknown".
export const postEnableRebranding = (organizationIds: string[]): Promise<{ enabled: number }> => {
  const loginStore = useLoginStore()

  return axios
    .post<EnableRebrandingResponse>(
      `${API_URL}/rebranding/organizations`,
      { organization_ids: organizationIds },
      { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } },
    )
    .then((res) => res.data.data)
}

export const patchDisableRebranding = (organizationLogtoId: string) => {
  const loginStore = useLoginStore()

  return axios.patch(
    `${API_URL}/rebranding/${organizationLogtoId}/disable`,
    {},
    { headers: { Authorization: `Bearer ${loginStore.jwtToken}` } },
  )
}
