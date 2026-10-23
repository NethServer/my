<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { useLatestInventory } from '@/queries/systems/latestInventory'
import {
  faEarthAmericas,
  faLocationDot,
  faNetworkWired,
  faShield,
  faStar,
  faUsers,
  faWifi,
} from '@fortawesome/free-solid-svg-icons'
import { computed } from 'vue'
import type { InventoryNetworkInterface, NsecFacts } from '@/lib/systems/inventory'
import type { Ns8Facts, Ns8NetworkInterface } from '@/lib/systems/ns8Facts'
import { compareNetworkZones, getNetworkInterfaceZone } from '@/lib/systems/nsecFacts'
import { netmaskToCIDR } from '@/lib/network'

// ns8 keeps network facts per cluster node, so the card is told which one to
// show; without a node id it reads the single nsec configuration.
const { nodeId } = defineProps<{
  nodeId?: string
}>()

const { state: latestInventory } = useLatestInventory()

const ns8Node = computed(() => {
  if (!nodeId) {
    return undefined
  }
  const facts = latestInventory.value.data?.data?.facts as Ns8Facts | undefined
  return facts?.nodes?.[nodeId]
})

const dnsServers = computed(() => {
  if (nodeId) {
    return ns8Node.value?.dns_servers || []
  }
  const facts = latestInventory.value.data?.data?.facts as NsecFacts | undefined
  return facts?.dns_servers || []
})

const networkInterfaces = computed((): (InventoryNetworkInterface | Ns8NetworkInterface)[] => {
  const facts = latestInventory.value.data?.data?.facts as NsecFacts | undefined
  const networkConfig = nodeId
    ? ns8Node.value?.network?.configuration
    : facts?.features?.network?.configuration
  if (!networkConfig) {
    return []
  }
  // by zone, then by the name shown first; numeric collation so eth2 comes
  // before eth10
  return Object.values(networkConfig).sort(
    (a, b) =>
      compareNetworkZones(getInterfaceZone(a), getInterfaceZone(b)) ||
      getDisplayName(a).localeCompare(getDisplayName(b), undefined, { numeric: true }),
  )
})

const getIpAddressWithCidr = (iface: InventoryNetworkInterface | Ns8NetworkInterface) => {
  const ipaddr = iface.props?.ipaddr || ''
  const netmask = iface.props?.netmask || ''

  if (ipaddr && netmask) {
    // calculate CIDR from netmask
    const cidr = netmaskToCIDR(netmask)
    return `${ipaddr}${cidr}`
  } else {
    return '-'
  }
}

const isNsecInterface = (
  iface: InventoryNetworkInterface | Ns8NetworkInterface,
): iface is InventoryNetworkInterface => 'zone' in iface.props || 'role' in iface.props

// ns8 places its interfaces in no firewall zone, so they get a neutral look of
// their own and the zone is left out of the subtitle
const getInterfaceZone = (iface: InventoryNetworkInterface | Ns8NetworkInterface) =>
  isNsecInterface(iface) ? getNetworkInterfaceZone(iface) : undefined

// only python3-nethsec >= 1.8.1 reports the logical interface; elsewhere the
// device name is all there is
const getLogicalInterfaceName = (iface: InventoryNetworkInterface | Ns8NetworkInterface) =>
  'interface' in iface.props ? iface.props.interface : undefined

const getDisplayName = (iface: InventoryNetworkInterface | Ns8NetworkInterface) =>
  getLogicalInterfaceName(iface) || iface.name

const getInterfaceIcon = (iface: InventoryNetworkInterface | Ns8NetworkInterface) => {
  const zone = getInterfaceZone(iface)
  return zone === undefined ? faNetworkWired : getNetworkZoneIcon(zone)
}

const getInterfaceBackgroundStyle = (iface: InventoryNetworkInterface | Ns8NetworkInterface) => {
  const zone = getInterfaceZone(iface)
  return zone === undefined
    ? 'bg-indigo-100 dark:bg-indigo-700'
    : getNetworkZoneBackgroundStyle(zone)
}

const getInterfaceForegroundStyle = (iface: InventoryNetworkInterface | Ns8NetworkInterface) => {
  const zone = getInterfaceZone(iface)
  return zone === undefined
    ? 'text-indigo-700 dark:text-indigo-50'
    : getNetworkZoneForegroundStyle(zone)
}

const getNetworkZoneIcon = (zone: string) => {
  switch (zone) {
    case 'lan':
      return faLocationDot
    case 'wan':
      return faEarthAmericas
    case 'guest':
      return faUsers
    case 'dmz':
      return faShield
    case 'hotspot':
      return faWifi
    default:
      return faStar
  }
}

const getNetworkZoneBackgroundStyle = (zone: string) => {
  switch (zone) {
    case 'lan':
      return 'bg-green-100 dark:bg-green-700'
    case 'wan':
      return 'bg-rose-100 dark:bg-rose-700'
    case 'guest':
      return 'bg-blue-100 dark:bg-blue-700'
    case 'dmz':
      return 'bg-amber-100 dark:bg-amber-700'
    case 'hotspot':
      return 'bg-sky-100 dark:bg-sky-700'
    default:
      return 'bg-violet-100 dark:bg-violet-700'
  }
}

const getNetworkZoneForegroundStyle = (zone: string) => {
  switch (zone) {
    case 'lan':
      return 'text-green-700 dark:text-green-50'
    case 'wan':
      return 'text-rose-700 dark:text-rose-50'
    case 'guest':
      return 'text-blue-700 dark:text-blue-50'
    case 'dmz':
      return 'text-amber-700 dark:text-amber-50'
    case 'hotspot':
      return 'text-sky-700 dark:text-sky-50'
    default:
      return 'text-violet-700 dark:text-violet-50'
  }
}
</script>

<template>
  <NeCard>
    <div class="mb-4 flex items-center gap-4">
      <FontAwesomeIcon :icon="faNetworkWired" class="size-5 shrink-0" aria-hidden="true" />
      <NeHeading tag="h6">
        {{ $t('system_detail.network').toUpperCase() }}
      </NeHeading>
    </div>
    <!-- get latest inventory error notification -->
    <NeInlineNotification
      v-if="latestInventory.status === 'error'"
      kind="error"
      :title="$t('system_detail.cannot_retrieve_latest_inventory')"
      :description="latestInventory.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="latestInventory.status === 'pending'" :lines="8" />
    <div v-else class="space-y-6">
      <!-- network interfaces -->
      <div v-if="networkInterfaces.length" class="mt-8 flex flex-wrap justify-center gap-16">
        <div
          class="flex flex-col items-center"
          v-for="iface in networkInterfaces"
          :key="iface.name"
        >
          <!-- icon -->
          <div
            :class="`flex size-16 shrink-0 items-center justify-center rounded-full ${getInterfaceBackgroundStyle(iface)}`"
          >
            <FontAwesomeIcon
              :icon="getInterfaceIcon(iface)"
              aria-hidden="true"
              :class="`size-8 ${getInterfaceForegroundStyle(iface)}`"
            />
          </div>
          <!-- name: the logical interface over its device, or the device alone -->
          <div class="mt-2 text-base font-medium">
            {{ getDisplayName(iface) }}
          </div>
          <div
            v-if="getLogicalInterfaceName(iface)"
            class="text-tertiary-neutral dark:text-tertiary-neutral"
          >
            {{ iface.name }}
          </div>
          <!-- type and zone -->
          <div class="text-tertiary-neutral dark:text-tertiary-neutral mt-1">
            {{ iface?.type || '-' }}
            <span v-if="getInterfaceZone(iface)"
              >&bull;
              <span class="uppercase">{{ getInterfaceZone(iface) }}</span>
            </span>
          </div>
          <!-- ip address -->
          <div class="text-tertiary-neutral dark:text-tertiary-neutral">
            {{ getIpAddressWithCidr(iface) }}
          </div>
          <!-- gateway -->
          <div
            v-if="iface.props?.gateway"
            class="text-tertiary-neutral dark:text-tertiary-neutral uppercase"
          >
            GW: {{ iface.props?.gateway }}
          </div>
        </div>
      </div>
      <div class="divide-y divide-gray-200 dark:divide-gray-700">
        <!-- dns -->
        <div class="flex gap-4 py-4">
          <span class="shrink-0 font-medium">
            {{ $t('system_detail.dns_servers') }}
          </span>
          <span v-if="dnsServers.length" class="text-tertiary-neutral dark:text-tertiary-neutral">
            {{ dnsServers.join(', ') }}
          </span>
          <span v-else>-</span>
        </div>
      </div>
    </div>
  </NeCard>
</template>
