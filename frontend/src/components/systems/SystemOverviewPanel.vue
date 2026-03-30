<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useSystemDetail } from '@/queries/systems/systemDetail'
import SystemApplicationsCard from './SystemApplicationsCard.vue'
import SystemInfoCard from './SystemInfoCard.vue'
import SystemNetworkCard from './SystemNetworkCard.vue'
import SystemStatusCard from './SystemStatusCard.vue'
import SystemSubscriptionCard from './SystemSubscriptionCard.vue'

const { state: systemDetail } = useSystemDetail()
</script>

<template>
  <div class="3xl:grid-cols-4 grid grid-cols-1 gap-x-6 gap-y-6 md:grid-cols-2">
    <SystemInfoCard />
    <SystemStatusCard />
    <SystemSubscriptionCard />
    <SystemApplicationsCard v-if="systemDetail.data?.type === 'ns8'" />
    <SystemNetworkCard
      v-if="systemDetail.data?.type === 'nsec'"
      class="3xl:col-span-4 md:col-span-2"
    />
  </div>
</template>
