<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeEmptyState,
  NeInlineNotification,
  NeListbox,
  NeSkeleton,
  type NeListboxOption,
} from '@nethesis/vue-components'
import { faServer } from '@fortawesome/free-solid-svg-icons'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useLatestInventory } from '@/queries/systems/latestInventory'
import type { Ns8Facts } from '@/lib/systems/ns8Facts'
import type { NsecFacts } from '@/lib/systems/nsecFacts'
import {
  type ClusterNode,
  type HardwareInfo,
  ns8NodeHardwareInfo,
  nsecHardwareInfo,
  sortedNs8Nodes,
} from '@/lib/systems/hardware'
import HardwareGeneralCard from './HardwareGeneralCard.vue'
import HardwareCpuBiosCard from './HardwareCpuBiosCard.vue'
import HardwareMemoryCard from './HardwareMemoryCard.vue'
import HardwareStorageCard from './HardwareStorageCard.vue'
import SystemNetworkCard from './SystemNetworkCard.vue'

const { t } = useI18n()
const { state: systemDetail } = useSystemDetail()
const { state: latestInventory } = useLatestInventory()

const systemType = computed(() => systemDetail.value.data?.type)

const nodes = computed<ClusterNode[]>(() => {
  if (systemType.value !== 'ns8') {
    return []
  }
  return sortedNs8Nodes(latestInventory.value.data?.data?.facts as Ns8Facts | undefined)
})

const getNodeName = (node: ClusterNode) => {
  if (node.ui_name) {
    return t('system_detail.node_name_with_label', { id: node.id, label: node.ui_name })
  }
  return t('system_detail.node_name', { id: node.id })
}

const nodeOptions = computed((): NeListboxOption[] =>
  nodes.value.map((node) => ({ id: node.id, label: getNodeName(node) })),
)

const selectedNodeId = ref('')

// The cluster leader is the node people look at first, and the inventory can
// arrive after the panel is mounted, so the default is set as soon as nodes load
watch(
  nodes,
  (clusterNodes) => {
    if (!clusterNodes.some((node) => node.id === selectedNodeId.value)) {
      selectedNodeId.value = clusterNodes[0]?.id || ''
    }
  },
  { immediate: true },
)

const hardware = computed<HardwareInfo | undefined>(() => {
  const inventory = latestInventory.value.data

  if (!inventory?.data?.facts) {
    return undefined
  }

  if (systemType.value === 'ns8') {
    const node = nodes.value.find(({ id }) => id === selectedNodeId.value)
    return node ? ns8NodeHardwareInfo(node) : undefined
  }
  return nsecHardwareInfo(inventory.data.facts as NsecFacts, inventory.data.uuid)
})

const hasMountpoints = computed(() => Object.keys(hardware.value?.mountpoints || {}).length > 0)
</script>

<template>
  <div>
    <!-- get latest inventory error notification -->
    <NeInlineNotification
      v-if="latestInventory.status === 'error'"
      kind="error"
      :title="$t('system_detail.cannot_retrieve_latest_inventory')"
      :description="latestInventory.error.message"
      class="mb-6"
    />
    <NeSkeleton
      v-else-if="latestInventory.status === 'pending' || systemDetail.status === 'pending'"
      :lines="10"
    />
    <div v-else-if="hardware">
      <!-- ns8 node selector -->
      <NeListbox
        v-if="nodeOptions.length"
        v-model="selectedNodeId"
        :label="$t('system_detail.viewing_data_for')"
        :options="nodeOptions"
        :no-options-label="$t('ne_combobox.no_options_label')"
        :optional-label="$t('common.optional')"
        options-panel-style="max-w-64 w-full"
        class="mb-6 max-w-64"
      />
      <div class="3xl:grid-cols-4 grid grid-cols-1 gap-x-6 gap-y-6 xl:grid-cols-2">
        <HardwareGeneralCard :hardware="hardware" />
        <HardwareCpuBiosCard :hardware="hardware" />
        <HardwareMemoryCard :hardware="hardware" />
        <HardwareStorageCard v-if="hasMountpoints" :hardware="hardware" />
        <SystemNetworkCard
          v-if="systemType === 'ns8'"
          :node-id="selectedNodeId"
          class="3xl:col-span-4 md:col-span-2"
        />
      </div>
    </div>
    <NeEmptyState
      v-else
      :title="$t('system_detail.no_hardware_info')"
      :icon="faServer"
      class="bg-white dark:bg-gray-950"
    />
  </div>
</template>
