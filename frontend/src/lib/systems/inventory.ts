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
  data: any //// improve typing
}

//// move specific facts to separate files?

// interface NsecFacts { ////
//   distro: {
//     name: string
//     version: string
//   }
//   memory: {
//     // swap: { ////
//     //   used_bytes: number
//     //   available_bytes: number
//     // }
//     // ...
//   }
//   features: NsecFeatures
// }

// interface NsecFeatures {
//   ha: {
//     vips: number
//     enabled: boolean
//   }
//   ui: {
//     luci: boolean
//     port443: boolean
//     port9090: boolean
//   }
//   dpi: {
//     rules: number
//     enabled: boolean
//   }
//   qos: {
//     count: number
//     rules: unknown[]
//   }
//   ddns: {
//     enabled: boolean
//   }
//   snmp: {
//     enabled: boolean
//   }
//   ipsec: {
//     count: number
//   }
//   snort: {
//     policy: string
//     enabled: boolean
//     oink_enabled: boolean
//     disabled_rules: number
//     bypass_dst_ipv4: number
//     bypass_dst_ipv6: number
//     bypass_src_ipv4: number
//     bypass_src_ipv6: number
//     suppressed_rules: number
//   }
//   adblock: {
//     enabled: boolean
//     community: number
//     enterprise: number
//   }
//   backups: {
//     passphrase_date: number
//     backup_passphrase: boolean
//   }
//   hotspot: {
//     server: string
//     enabled: boolean
//     interface: string
//   }
//   netifyd: {
//     enabled: boolean
//   }
//   network: NsecNetworkFeature
// }

// interface NsecNetworkFeature {
//   zones: {
//     ipv4: number
//     ipv6: number
//     name: string
//   }[]
//   route_info: {
//     count_ipv4_route: number
//     count_ipv6_route: number
//   }
//   interface_counts: {
//     bonds: number
//     vlans: number
//     bridges: number
//   }
//   zone_network_counts: Record<string, number>
// }

//// remove?
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
