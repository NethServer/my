//  Copyright (C) 2025 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import axios from 'axios'
import { API_URL } from '../config'
import { useLoginStore } from '@/stores/login'
import type { NsecFacts } from './nsecFacts'
import type { Ns8Facts } from './ns8Facts'

export type { NsecFacts, NsecFeatures, NsecNetworkFeature } from './nsecFacts'

export interface PciDevice {
  class_id: string
  vendor_id: string
  device_id: string
  revision: string
  class_name: string
  vendor_name: string
  device_name: string
  driver: string
}

export interface Distro {
  name: string
  version: string
}

export interface Memory {
  swap: {
    used_bytes: number
    available_bytes: number
  }
  system: {
    used_bytes: number
    available_bytes: number
  }
}

export interface Product {
  name: string
  manufacturer: string
}

export interface Processors {
  count: number | string
  model: string
  architecture: string
}

export const LATEST_INVENTORY_KEY = 'latestInventory'

interface InventoryData {
  id: number
  system_id: string
  timestamp: string
  data: {
    uuid: string
    installation: string
    $schema: string
    facts: Ns8Facts | NsecFacts
  }
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
