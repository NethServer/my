//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import { describe, expect, it } from 'vitest'
import { compareNetworkZones, getNetworkInterfaceZone } from './nsecFacts'
import type { CurrentInventoryNetworkInterface, LegacyInventoryNetworkInterface } from './inventory'

const addressing = { ipaddr: '10.10.40.1', netmask: '255.255.255.0', gateway: null }

const legacy = (role: string | null): LegacyInventoryNetworkInterface => ({
  name: 'eth2.40',
  type: 'ethernet',
  props: { ...addressing, role },
})

const current = (zone: string | null): CurrentInventoryNetworkInterface => ({
  name: 'eth2.40',
  type: 'ethernet',
  props: { ...addressing, interface: 'voip', zone },
})

describe('getNetworkInterfaceZone', () => {
  it('should map the legacy green and red roles to the lan and wan zones', () => {
    expect(getNetworkInterfaceZone(legacy('green'))).toBe('lan')
    expect(getNetworkInterfaceZone(legacy('red'))).toBe('wan')
  })

  it('should take any other legacy role as the zone name', () => {
    expect(getNetworkInterfaceZone(legacy('guest'))).toBe('guest')
    expect(getNetworkInterfaceZone(legacy('voip'))).toBe('voip')
  })

  it('should return the zone of the current format as it is', () => {
    expect(getNetworkInterfaceZone(current('lan'))).toBe('lan')
    expect(getNetworkInterfaceZone(current('voip'))).toBe('voip')
  })

  it('should return undefined for an interface outside every zone', () => {
    expect(getNetworkInterfaceZone(legacy(null))).toBeUndefined()
    expect(getNetworkInterfaceZone(current(null))).toBeUndefined()
  })
})

describe('compareNetworkZones', () => {
  it('should put built-in zones in the NethSecurity UI order, then custom zones by name, then no zone', () => {
    const zones = [undefined, 'voip', 'ipsec', 'dmz', 'iot', 'wan', 'hotspot', 'lan', 'guest']
    expect(zones.sort(compareNetworkZones)).toEqual([
      'lan',
      'wan',
      'guest',
      'dmz',
      'hotspot',
      'ipsec',
      'iot',
      'voip',
      undefined,
    ])
  })

  // sort() moves undefined to the end without calling the comparator
  it('should put an interface outside every zone after any zone', () => {
    expect(compareNetworkZones('voip', undefined)).toBeLessThan(0)
    expect(compareNetworkZones(undefined, 'lan')).toBeGreaterThan(0)
  })

  it('should treat the same zone as equal', () => {
    expect(compareNetworkZones('lan', 'lan')).toBe(0)
    expect(compareNetworkZones('voip', 'voip')).toBe(0)
    expect(compareNetworkZones(undefined, undefined)).toBe(0)
  })
})
