<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import CounterCard from '@/components/CounterCard.vue'
import { useInventoryChanges } from '@/queries/systems/inventoryChanges'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useI18n } from 'vue-i18n'
import SystemApplicationsCard from './SystemApplicationsCard.vue'
import SystemInfoCard from './SystemInfoCard.vue'
import SystemNetworkCard from './SystemNetworkCard.vue'
import SystemStatusCard from './SystemStatusCard.vue'
import SystemSubscriptionCard from './SystemSubscriptionCard.vue'
import { NeHeading } from '@nethesis/vue-components'

const { t } = useI18n()
const { state: systemDetail } = useSystemDetail()
const { state: changesState } = useInventoryChanges()
</script>

<template>
  <div class="3xl:grid-cols-4 grid grid-cols-1 gap-x-6 gap-y-6 md:grid-cols-2">
    <SystemInfoCard />
    <SystemStatusCard />
    <SystemSubscriptionCard />
    <SystemApplicationsCard v-if="systemDetail.data?.type === 'ns8'" />
    <SystemNetworkCard class="3xl:col-span-4 md:col-span-2" />
  </div>
  <NeHeading tag="h5" class="mt-9 mb-5">
    {{ $t('system_detail.change_history') }}
  </NeHeading>
  <!-- change counters -->
  <div class="mt-6 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
    <CounterCard
      :title="t('system_detail.total_changes')"
      :counter="changesState.data?.total_changes ?? 0"
      :loading="changesState.status === 'pending'"
    />
    <CounterCard
      :title="t('system_detail.critical_changes')"
      :counter="changesState.data?.changes_by_severity?.critical ?? 0"
      :loading="changesState.status === 'pending'"
      color-classes="text-rose-700 dark:text-rose-500"
    />
    <CounterCard
      :title="t('system_detail.high_changes')"
      :counter="changesState.data?.changes_by_severity?.high ?? 0"
      :loading="changesState.status === 'pending'"
      color-classes="text-orange-600 dark:text-orange-500"
    />
    <CounterCard
      :title="t('system_detail.medium_changes')"
      :counter="changesState.data?.changes_by_severity?.medium ?? 0"
      :loading="changesState.status === 'pending'"
      color-classes="text-yellow-600 dark:text-yellow-700"
    />
  </div>
</template>
