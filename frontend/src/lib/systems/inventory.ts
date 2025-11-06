//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'

export const LATEST_INVENTORY_KEY = 'latestInventory'

interface InventoryData {
  id: number
  system_id: string
  timestamp: string
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  data: any // The structure of inventory data can be complex and varied
}

export interface EsmithConfiguration {
  name: string
  type: string
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  props: any
}

interface LatestInventoryResponse {
  code: number
  message: string
  data: InventoryData
}

export interface InventoryNetworkInterface {
  name: string
  type: string
  props: {
    role: string
    ipaddr: string
    gateway: string | null
    netmask: string
  }
}

export const getLatestInventory = (systemId: string) => {
  const loginStore = useLoginStore()

  return axios
    .get<LatestInventoryResponse>(`${API_URL}/systems/${systemId}/inventory/latest`, {
      headers: { Authorization: `Bearer ${loginStore.jwtToken}` },
    })
    .then((res) => res.data.data)
}
