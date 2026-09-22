<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeBadgeV2,
  NeCard,
  NeEmptyState,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
} from '@nethesis/vue-components'
import type { NeBadgeV2Kind } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faMicrochip } from '@fortawesome/free-solid-svg-icons'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useLatestInventory } from '@/queries/systems/latestInventory'
import type { Ns8Facts } from '@/lib/systems/ns8Facts'
import { type ClusterNode, ns8NodeName, sortedNs8Nodes } from '@/lib/systems/hardware'

const { t } = useI18n()
const { state: latestInventory } = useLatestInventory()

const nodes = computed<ClusterNode[]>(() =>
  sortedNs8Nodes(latestInventory.value.data?.data?.facts as Ns8Facts | undefined),
)

const getNodeRole = (node: ClusterNode) => {
  return node.cluster_leader
    ? t('system_detail.node_role_leader')
    : t('system_detail.node_role_worker')
}

const getNodeBadgeKind = (node: ClusterNode): NeBadgeV2Kind => {
  return node.cluster_leader ? 'green' : 'indigo'
}

const getNodeBackgroundStyle = (node: ClusterNode) => {
  return node.cluster_leader ? 'bg-green-100 dark:bg-green-700' : 'bg-indigo-100 dark:bg-indigo-700'
}

const getNodeForegroundStyle = (node: ClusterNode) => {
  return node.cluster_leader
    ? 'text-green-700 dark:text-green-50'
    : 'text-indigo-700 dark:text-indigo-50'
}
</script>

<template>
  <NeCard>
    <div class="mb-4 flex items-center gap-4">
      <FontAwesomeIcon :icon="faMicrochip" class="size-5 shrink-0" aria-hidden="true" />
      <NeHeading tag="h6">
        {{ $t('system_detail.cluster_nodes').toUpperCase() }}
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
    <!-- cluster nodes -->
    <div v-else-if="nodes.length" class="mt-8 mb-6 flex flex-wrap justify-center gap-16">
      <div class="flex flex-col items-center" v-for="node in nodes" :key="node.id">
        <!-- icon -->
        <div
          :class="`flex size-16 shrink-0 items-center justify-center rounded-full ${getNodeBackgroundStyle(node)}`"
        >
          <FontAwesomeIcon
            :icon="faMicrochip"
            aria-hidden="true"
            :class="`size-8 ${getNodeForegroundStyle(node)}`"
          />
        </div>
        <!-- name -->
        <div class="mt-2 text-base font-medium">
          {{ ns8NodeName(node, t) }}
        </div>
        <!-- fqdn -->
        <div class="text-tertiary-neutral dark:text-tertiary-neutral mt-1">
          {{ node.fqdn || '-' }}
        </div>
        <!-- role -->
        <NeBadgeV2 :kind="getNodeBadgeKind(node)" size="xs" class="mt-2">
          {{ getNodeRole(node) }}
        </NeBadgeV2>
      </div>
    </div>
    <NeEmptyState
      v-else
      :title="$t('system_detail.no_nodes')"
      :icon="faMicrochip"
      class="bg-white dark:bg-gray-950"
    />
  </NeCard>
</template>
