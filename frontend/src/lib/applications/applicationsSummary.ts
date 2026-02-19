//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import * as v from 'valibot'

export const APPLICATIONS_SUMMARY_KEY = 'applicationsSummary'

export const ApplicationTypeSchema = v.object({
  instance_of: v.string(),
  name: v.string(),
  count: v.number(),
})

export type ApplicationType = v.InferOutput<typeof ApplicationTypeSchema>

export const ApplicationsSummaryDataSchema = v.object({
  total: v.number(),
  by_type: v.array(ApplicationTypeSchema),
})

export type ApplicationsSummaryData = v.InferOutput<typeof ApplicationsSummaryDataSchema>

interface ApplicationsSummaryResponse {
  code: number
  message: string
  data: ApplicationsSummaryData
}

export const getApplicationsSummary = (
  companyId: string,
  page: number,
  pageSize: number,
  sortBy: string,
  sortDescending: boolean,
) => {
  const loginStore = useLoginStore()

  const params = new URLSearchParams({
    organization_id: companyId,
    page: page.toString(),
    page_size: pageSize.toString(),
    sort_by: sortBy,
    sort_direction: sortDescending ? 'desc' : 'asc',
  })
  const queryString = params.toString()
  const url = `${API_URL}/applications/summary?${queryString}`

  return axios
    .get<ApplicationsSummaryResponse>(url, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
