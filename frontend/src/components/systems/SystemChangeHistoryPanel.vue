<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import CounterCard from '@/components/CounterCard.vue'
import { useInventoryChanges } from '@/queries/systems/inventoryChanges'
import { useI18n } from 'vue-i18n'
import UpdatingSpinner from '../UpdatingSpinner.vue'
import { NeInlineNotification } from '@nethesis/vue-components'

const { t } = useI18n()

const { state: inventoryChanges, asyncStatus: inventoryChangesAsyncStatus } = useInventoryChanges()
</script>

<template>
  <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
    <div class="max-w-2xl text-gray-500 dark:text-gray-400">
      {{ $t('system_detail.change_history_description') }}
    </div>
    <!-- update indicator -->
    <UpdatingSpinner
      v-if="inventoryChangesAsyncStatus === 'loading' && inventoryChanges.status !== 'pending'"
    />
  </div>
  <!-- inventory changes error notification -->
  <NeInlineNotification
    v-if="inventoryChanges.status === 'error'"
    kind="error"
    :title="t('system_detail.cannot_retrieve_latest_inventory')"
    :description="inventoryChanges.error.message"
    class="mb-6"
  />
  <!-- change counters -->
  <div class="grid grid-cols-2 gap-6 sm:grid-cols-4">
    <CounterCard
      :title="t('system_detail.total_changes')"
      :counter="inventoryChanges.data?.total_changes ?? 0"
      :loading="inventoryChanges.status === 'pending'"
    />
    <CounterCard
      :title="t('system_detail.critical_changes')"
      :counter="inventoryChanges.data?.changes_by_severity?.critical ?? 0"
      :loading="inventoryChanges.status === 'pending'"
      color-classes="text-rose-700 dark:text-rose-500"
    />
    <CounterCard
      :title="t('system_detail.high_changes')"
      :counter="inventoryChanges.data?.changes_by_severity?.high ?? 0"
      :loading="inventoryChanges.status === 'pending'"
      color-classes="text-orange-600 dark:text-orange-500"
    />
    <CounterCard
      :title="t('system_detail.medium_changes')"
      :counter="inventoryChanges.data?.changes_by_severity?.medium ?? 0"
      :loading="inventoryChanges.status === 'pending'"
      color-classes="text-yellow-700 dark:text-yellow-400"
    />
  </div>
</template>
