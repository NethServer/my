//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import { type Pagination } from '../common'

export const INVENTORY_DIFFS_KEY = 'inventoryDiffs'
export const INVENTORY_DIFFS_TABLE_ID = 'inventoryDiffsTable'

export type InventoryDiffSeverity = 'low' | 'medium' | 'high' | 'critical'

export type InventoryDiffCategory =
  | 'os'
  | 'hardware'
  | 'network'
  | 'security'
  | 'backup'
  | 'features'
  | 'modules'
  | 'cluster'
  | 'nodes'
  | 'system'

export type InventoryDiffType = 'create' | 'update' | 'delete'

export interface InventoryDiff {
  id: number
  system_id: string
  previous_inventory_id: number | null
  inventory_id: number
  diff_type: InventoryDiffType
  field_path: string
  previous_value: unknown
  current_value: unknown
  severity: InventoryDiffSeverity
  category: InventoryDiffCategory
  notification_sent: boolean
  created_at: string
}

interface InventoryDiffsResponse {
  code: number
  message: string
  data: {
    diffs: InventoryDiff[]
    pagination: Pagination
  }
}

const getInventoryDiffsQueryStringParams = (
  pageNum: number,
  pageSize: number,
  severity: InventoryDiffSeverity[],
  category: InventoryDiffCategory[],
  diffType: InventoryDiffType[],
  inventoryId: number[],
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
  inventoryId.forEach((id) => searchParams.append('inventory_id', id.toString()))

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

export const getInventoryDiffs = (
  systemId: string,
  pageNum: number,
  pageSize: number,
  severity: InventoryDiffSeverity[],
  category: InventoryDiffCategory[],
  diffType: InventoryDiffType[],
  inventoryId: number[],
  fromDate: string,
  toDate: string,
  search: string,
) => {
  const loginStore = useLoginStore()
  const queryString = getInventoryDiffsQueryStringParams(
    pageNum,
    pageSize,
    severity,
    category,
    diffType,
    inventoryId,
    fromDate,
    toDate,
    search,
  )

  return axios
    .get<InventoryDiffsResponse>(`${API_URL}/systems/${systemId}/inventory/diffs?${queryString}`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
