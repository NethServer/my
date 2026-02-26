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
  faEthernet,
  faLocationDot,
  faNetworkWired,
  faShield,
  faStar,
  faUsers,
  faWifi,
} from '@fortawesome/free-solid-svg-icons'
import { computed } from 'vue'
import type { EsmithConfiguration, InventoryNetworkInterface } from '@/lib/systems/inventory'
import { netmaskToCIDR } from '@/lib/network'

const { state: latestInventory } = useLatestInventory()

const dnsServers = computed(() => {
  const esmithConfig = latestInventory.value.data?.data?.esmithdb?.configuration || []
  const dnsEntry = esmithConfig.find((entry: EsmithConfiguration) => entry.name === 'dns')
  const dnsServers = dnsEntry?.props?.NameServers || ''

  if (!dnsServers) {
    return []
  } else {
    return dnsServers.split(',')
  }
})

const networkInterfaces = computed(() => {
  return latestInventory.value.data?.data?.esmithdb?.networks || []
})

const getIpAddressWithCidr = (iface: InventoryNetworkInterface) => {
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

const isNethsecurity = computed(() => {
  return latestInventory.value.data?.data?.os?.type === 'nethsecurity'
})

const getNetworkRoleIcon = (role: string | undefined) => {
  if (!isNethsecurity.value) {
    return faEthernet
  }

  switch (role) {
    case 'green':
      return faLocationDot
    case 'red':
      return faEarthAmericas
    case 'blue':
      return faUsers
    case 'orange':
      return faShield
    case 'hotspot':
      return faWifi
    default:
      return faStar
  }
}

const getNetworkRoleBackgroundStyle = (role: string | undefined) => {
  if (!isNethsecurity.value) {
    return 'bg-gray-100 dark:bg-gray-700'
  }

  switch (role) {
    case 'green':
      return 'bg-green-100 dark:bg-green-700'
    case 'red':
      return 'bg-rose-100 dark:bg-rose-700'
    case 'blue':
      return 'bg-blue-100 dark:bg-blue-700'
    case 'orange':
      return 'bg-amber-100 dark:bg-amber-700'
    case 'hotspot':
      return 'bg-sky-100 dark:bg-sky-700'
    default:
      return 'bg-violet-100 dark:bg-violet-700'
  }
}

const getNetworkRoleForegroundStyle = (role: string | undefined) => {
  if (!isNethsecurity.value) {
    return 'text-gray-700 dark:text-gray-300'
  }

  switch (role) {
    case 'green':
      return 'text-green-700 dark:text-green-50'
    case 'red':
      return 'text-rose-700 dark:text-rose-50'
    case 'blue':
      return 'text-blue-700 dark:text-blue-50'
    case 'orange':
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
            :class="`flex size-16 flex-shrink-0 items-center justify-center rounded-full ${getNetworkRoleBackgroundStyle(iface.props?.role)}`"
          >
            <FontAwesomeIcon
              :icon="getNetworkRoleIcon(iface.props?.role)"
              aria-hidden="true"
              :class="`size-8 ${getNetworkRoleForegroundStyle(iface.props?.role)}`"
            />
          </div>
          <!-- name -->
          <div class="mt-2 text-lg font-medium">
            {{ iface.name }}
          </div>
          <!-- type and role -->
          <div class="text-gray-600 dark:text-gray-300">
            {{ iface?.type || '-' }}
            <span v-if="latestInventory.data.data?.os?.type === 'nethsecurity'"
              >&bull;
              {{ iface.props?.role || '-' }}
            </span>
          </div>
          <!-- ip address -->
          <div class="text-gray-600 dark:text-gray-300">
            {{ getIpAddressWithCidr(iface) }}
          </div>
          <!-- gateway -->
          <div v-if="iface.props?.gateway" class="text-gray-600 uppercase dark:text-gray-300">
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
          <span v-if="dnsServers.length" class="text-gray-600 dark:text-gray-300">
            {{ dnsServers.join(', ') }}
          </span>
          <span v-else>-</span>
        </div>
      </div>
    </div>
  </NeCard>
</template>
