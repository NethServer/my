//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import { type Pagination } from '../common'
import {
  type InventoryDiffCategory,
  type InventoryDiffSeverity,
  type InventoryDiffType,
} from './inventoryDiffs'

export const INVENTORY_TIMELINE_KEY = 'inventoryTimeline'
export const INVENTORY_TIMELINE_TABLE_ID = 'inventoryTimelineTable'

export interface InventoryTimelineSummary {
  total: number
  critical: number
  high: number
  medium: number
  low: number
}

export interface InventoryTimelineGroup {
  date: string
  inventory_count: number
  change_count: number
  inventory_ids: number[]
}

interface InventoryTimelineResponse {
  code: number
  message: string
  data: {
    summary: InventoryTimelineSummary
    groups: InventoryTimelineGroup[]
    pagination: Pagination
  }
}

const getInventoryTimelineQueryStringParams = (
  pageNum: number,
  pageSize: number,
  severity: InventoryDiffSeverity[],
  category: InventoryDiffCategory[],
  diffType: InventoryDiffType[],
  fromDate: string,
  toDate: string,
  search: string,
) => {
  const searchParams = new URLSearchParams({
    page: pageNum.toString(),
    page_size: pageSize.toString(),
  })

  severity.forEach((s) => searchParams.append('severity', s))
  category.forEach((c) => searchParams.append('category', c))
  diffType.forEach((d) => searchParams.append('diff_type', d))

  if (fromDate.trim()) {
    searchParams.append('from_date', fromDate)
  }

  if (toDate.trim()) {
    searchParams.append('to_date', toDate)
  }

  if (search.trim()) {
    searchParams.append('search', search)
  }

  return searchParams.toString()
}

export const getInventoryTimeline = (
  systemId: string,
  pageNum: number,
  pageSize: number,
  severity: InventoryDiffSeverity[],
  category: InventoryDiffCategory[],
  diffType: InventoryDiffType[],
  fromDate: string,
  toDate: string,
  search: string,
) => {
  const loginStore = useLoginStore()
  const queryString = getInventoryTimelineQueryStringParams(
    pageNum,
    pageSize,
    severity,
    category,
    diffType,
    fromDate,
    toDate,
    search,
  )

  return axios
    .get<InventoryTimelineResponse>(
      `${API_URL}/systems/${systemId}/inventory/timeline?${queryString}`,
      {
        headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
      },
    )
    .then((res) => res.data.data)
}
