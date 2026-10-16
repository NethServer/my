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
  ns8NodeName,
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

// A single-node cluster has nothing to choose between
const showNodeMenu = computed(() => nodes.value.length > 1)

const nodeOptions = computed((): NeListboxOption[] =>
  nodes.value.map((node) => ({ id: node.id, label: ns8NodeName(node, t) })),
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
    <div v-else-if="hardware" class="flex flex-col gap-6 lg:flex-row lg:items-start">
      <!-- ns8 node selector: a long node list would push the cards off a small screen -->
      <NeListbox
        v-if="showNodeMenu"
        v-model="selectedNodeId"
        :label="$t('system_detail.viewing_data_for')"
        :options="nodeOptions"
        :no-options-label="$t('ne_combobox.no_options_label')"
        :optional-label="$t('common.optional')"
        options-panel-style="max-w-64 w-full"
        class="max-w-64 lg:hidden"
      />
      <!-- ns8 node menu: hover and selection mirror the side menu -->
      <nav
        v-if="showNodeMenu"
        :aria-label="$t('system_detail.cluster_nodes')"
        class="hidden lg:block lg:w-56 lg:shrink-0"
      >
        <p class="text-tertiary-neutral px-3 pb-2 text-sm leading-4 font-normal uppercase">
          {{ $t('system_detail.cluster_nodes') }}
        </p>
        <ul role="list" class="mt-0.5 space-y-1">
          <li v-for="node in nodes" :key="node.id">
            <button
              type="button"
              :aria-current="node.id === selectedNodeId ? 'true' : undefined"
              :class="[
                node.id === selectedNodeId
                  ? 'border-primary-700 dark:border-primary-500 bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-50'
                  : 'text-tertiary-neutral dark:text-tertiary-neutral border-transparent hover:text-gray-900 dark:hover:text-gray-50',
                'flex w-full cursor-pointer items-center rounded-md border-l-4 px-3 py-2 text-left text-sm leading-6 font-semibold wrap-anywhere transition-colors duration-(--duration-small) hover:bg-gray-100 dark:hover:bg-gray-800',
              ]"
              @click="selectedNodeId = node.id"
            >
              {{ ns8NodeName(node, t) }}
            </button>
          </li>
        </ul>
      </nav>
      <!--
        The node menu takes about one breakpoint of width, so with it the
        columns step up one breakpoint later to keep the cards as wide
      -->
      <div
        :class="[
          'grid min-w-0 flex-1 grid-cols-1 gap-x-6 gap-y-6',
          showNodeMenu ? '4xl:grid-cols-4 2xl:grid-cols-2' : '3xl:grid-cols-4 xl:grid-cols-2',
        ]"
      >
        <HardwareGeneralCard :hardware="hardware" />
        <HardwareCpuBiosCard :hardware="hardware" />
        <HardwareMemoryCard :hardware="hardware" />
        <HardwareStorageCard v-if="hasMountpoints" :hardware="hardware" />
        <SystemNetworkCard
          v-if="systemType === 'ns8'"
          :node-id="selectedNodeId"
          :class="showNodeMenu ? '4xl:col-span-4 2xl:col-span-2' : '3xl:col-span-4 xl:col-span-2'"
        />
      </div>
    </div>
    <NeEmptyState
      v-else
      :title="$t('system_detail.no_hardware_info')"
      :description="$t('system_detail.no_hardware_info_description')"
      :icon="faServer"
      class="bg-white dark:bg-gray-950"
    />
  </div>
</template>
