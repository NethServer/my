//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type {
  InventoryNetworkInterface,
  PciDevice,
  Distro,
  Memory,
  Product,
  Processors,
} from './inventory'

export type { PciDevice }

export interface NsecFacts {
  pci: PciDevice[]
  fqdn: string
  distro: Distro
  memory: Memory
  product: Product
  virtual: string
  features: NsecFeatures
  timezone: string
  processors: Processors
  dns_servers: string[]
  mountpoints: Record<string, { used_bytes: number; total_bytes: number; available_bytes: number }>
  default_ipv4: string
  default_ipv6: string
  kernel_version: string
  uptime_seconds: number
  image_updates_available: boolean
  package_updates_available: boolean
}

export interface NsecFeatures {
  ha: {
    vips: number
    enabled: boolean
  }
  ui: {
    luci: boolean
    port443: boolean
    port9090: boolean
  }
  dpi: {
    rules: number
    enabled: boolean
  }
  qos: {
    count: number
    rules: {
      upload: number
      enabled: boolean
      download: number
    }[]
  }
  ddns: {
    enabled: boolean
  }
  snmp: {
    enabled: boolean
  }
  ipsec: {
    count: number
  }
  snort: {
    policy: string
    enabled: boolean
    oink_enabled: boolean
    disabled_rules: number
    bypass_dst_ipv4: number
    bypass_dst_ipv6: number
    bypass_src_ipv4: number
    bypass_src_ipv6: number
    suppressed_rules: number
  }
  adblock: {
    enabled: boolean
    community: number
    enterprise: number
  }
  backups: {
    passphrase_date: number
    backup_passphrase: boolean
  }
  hotspot: {
    server: string
    enabled: boolean
    interface: string
  }
  netifyd: {
    enabled: boolean
  }
  network: NsecNetworkFeature
  storage: {
    enabled: boolean
  }
  multiwan: {
    rules: number
    enabled: boolean
    policies: {
      backup: number
      custom: number
      balance: number
    }
  }
  wireguard: {
    enabled: boolean
    servers: Record<string, unknown>
  }
  controller: {
    enabled: boolean
  }
  flashstart: {
    bypass: number
    enabled: boolean
    pro_plus: boolean
    custom_servers: number
  }
  nathelpers: {
    count: number
    enabled: boolean
  }
  openvpn_rw: {
    server: number
    enabled: number
    instances: unknown[]
  }
  proxy_pass: {
    count: number
  }
  dhcp_server: {
    count: number
    static_leases: number
    dynamic_leases: number
    dns_records_count: number
    dns_forwarder_enabled: boolean
  }
  openvpn_tun: {
    client: number
    server: number
    tunnels: unknown[]
  }
  threat_shield: {
    enabled: boolean
    community: number
    enterprise: number
  }
  database_stats: {
    main: {
      users: number
    }
  }
  firewall_stats: {
    objects: {
      hosts: number
      rules: {
        input: number
        output: number
        forward: number
      }
      domains: number
      mwan_rules: number
      port_forward: {
        allowed_from: number
        destination_to: number
      }
    }
    firewall: {
      nat: {
        snat: number
        accept: number
        masquerade: number
      }
      rules: {
        input: number
        output: number
        forward: number
      }
      netmap: {
        source: number
        destination: number
      }
      port_forward: number
    }
  }
  mac_ip_binding: {
    disabled: number
    'hard-binding': number
    'soft-binding': number
  }
  default_password: {
    default_password: boolean
  }
  certificates_info: {
    acme_certificates: {
      count: number
      issued: number
      pending: number
    }
    custom_certificates: {
      count: number
    }
  }
  subscription_status: {
    status: string
  }
}

export interface NsecNetworkFeature {
  zones: {
    ipv4: number
    ipv6: number
    name: string
  }[]
  route_info: {
    count_ipv4_route: number
    count_ipv6_route: number
  }
  configuration: Record<string, InventoryNetworkInterface>
  interface_counts: {
    bonds: number
    vlans: number
    bridges: number
  }
  zone_network_counts: Record<string, number>
}
