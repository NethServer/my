<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import SystemAddonsCard from './SystemAddonsCard.vue'
import SystemApplicationsCard from './SystemApplicationsCard.vue'
import SystemInfoCard from './SystemInfoCard.vue'
import SystemNetworkCard from './SystemNetworkCard.vue'
import SystemStatusCard from './SystemStatusCard.vue'
import SystemSubscriptionCard from './SystemSubscriptionCard.vue'
import ClusterNodesCard from './ClusterNodesCard.vue'
import ClusterAddonsCard from './ClusterAddonsCard.vue'
import { canReadAddons } from '@/lib/permissions'

const { state: systemDetail } = useSystemDetail()

const isClusterAddonsCardShown = computed(
  () => systemDetail.value.data?.type === 'ns8' && canReadAddons(),
)
</script>

<template>
  <div class="3xl:grid-cols-4 grid grid-cols-1 gap-x-6 gap-y-6 xl:grid-cols-2">
    <SystemInfoCard />
    <SystemStatusCard />
    <SystemSubscriptionCard />
    <SystemApplicationsCard v-if="systemDetail.data?.type === 'ns8'" />
    <SystemAddonsCard v-if="systemDetail.data?.type === 'nsec'" />
    <!-- the cluster card reads the add-on grants, which need read:entitlements -->
    <ClusterAddonsCard v-if="isClusterAddonsCardShown" />
    <SystemNetworkCard
      v-if="systemDetail.data?.type === 'nsec'"
      class="3xl:col-span-4 xl:col-span-2"
    />
    <!-- beside the add-ons card when there is one, across the row otherwise -->
    <ClusterNodesCard
      v-if="systemDetail.data?.type === 'ns8'"
      :class="isClusterAddonsCardShown ? '3xl:col-span-3' : '3xl:col-span-4 xl:col-span-2'"
    />
  </div>
</template>
