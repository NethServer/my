//  Copyright (C) 2026 Nethesis S.r.l.
//  SPDX-License-Identifier: GPL-3.0-or-later

import type { Bios, Distro, Memory, Mountpoint, Processors } from './inventory'
import type { NsecFacts } from './nsecFacts'
import type { Ns8Facts, Ns8NodeFacts } from './ns8Facts'

/**
 * Hardware facts of a single machine: an nsec appliance, or one node of an ns8
 * cluster. Fields the two agents report differently are optional, so the cards
 * can simply skip the rows they have no data for.
 */
export interface HardwareInfo {
  productName: string
  manufacturer: string
  // '' on bare metal, otherwise the hypervisor, e.g. 'kvm' or 'Microsoft'
  virtual: string
  distro: Distro
  kernelVersion: string
  uuid: string
  processors: Processors
  memory: Memory
  bios?: Bios
  // ns8 only: for nsec these are already shown by the overview panel
  uptimeSeconds?: number
  timezone?: string
  mountpoints: Record<string, Mountpoint>
}

export type ClusterNode = Ns8NodeFacts & { id: string }

// Leader first, then workers with numeric collation so node 2 comes before node 10
export const sortedNs8Nodes = (facts: Ns8Facts | undefined): ClusterNode[] => {
  const nodesById = facts?.nodes

  if (!nodesById) {
    return []
  }
  return Object.entries(nodesById)
    .map(([id, node]) => ({ ...node, id }))
    .sort((a, b) => {
      if (a.cluster_leader !== b.cluster_leader) {
        return a.cluster_leader ? -1 : 1
      }
      return a.id.localeCompare(b.id, undefined, { numeric: true })
    })
}

export const nsecHardwareInfo = (facts: NsecFacts, uuid: string): HardwareInfo => ({
  productName: facts.product?.name || '',
  manufacturer: facts.product?.manufacturer || '',
  virtual: facts.virtual || '',
  distro: facts.distro,
  kernelVersion: facts.kernel_version,
  uuid,
  processors: facts.processors,
  memory: facts.memory,
  bios: facts.product?.bios,
  mountpoints: facts.mountpoints,
})

export const ns8NodeHardwareInfo = (node: Ns8NodeFacts): HardwareInfo => ({
  productName: node.product?.name || '',
  manufacturer: node.product?.manufacturer || '',
  virtual: node.virtual || '',
  distro: node.distro,
  kernelVersion: node.kernel_version,
  uuid: node.product?.uuid || '',
  processors: node.processors,
  memory: node.memory,
  bios: node.product?.bios,
  mountpoints: node.mountpoints,
  uptimeSeconds: node.uptime_seconds,
  timezone: node.timezone,
})

export const usagePercentage = (used: number, total: number): number => {
  if (!total) {
    return 0
  }
  return Math.round((used / total) * 100)
}

// NeProgressBar's color union is not re-exported by the package root
export type UsageBarColor = 'primary' | 'amber' | 'rose'

export const usageBarColor = (percentage: number): UsageBarColor => {
  if (percentage >= 90) {
    return 'rose'
  }
  if (percentage >= 70) {
    return 'amber'
  }
  return 'primary'
}
