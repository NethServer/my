//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import { type InventoryDiffCategory, type InventoryDiffSeverity } from './inventoryDiffs'

export const INVENTORY_CHANGES_KEY = 'inventoryChanges'

export interface InventoryChanges {
  system_id: string
  total_changes: number
  recent_changes: number
  last_inventory_time: string
  has_critical_changes: boolean
  has_alerts: boolean
  changes_by_category: Partial<Record<InventoryDiffCategory, number>>
  changes_by_severity: Partial<Record<InventoryDiffSeverity, number>>
}

interface InventoryChangesResponse {
  code: number
  message: string
  data: InventoryChanges | null
}

export const getInventoryChanges = (systemId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<InventoryChangesResponse>(`${API_URL}/systems/${systemId}/inventory/changes`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
