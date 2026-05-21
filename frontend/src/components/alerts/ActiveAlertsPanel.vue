<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import CounterCard from '@/components/CounterCard.vue'
import AlertsTable from '@/components/alerts/AlertsTable.vue'
import { useAlertsTotals } from '@/queries/alerts/alertsTotals'

const { t } = useI18n()

const { state: totalsState } = useAlertsTotals()

const totals = computed(() => totalsState.value?.data)
const isLoading = computed(() => totalsState.value?.status === 'pending')

const totalAlerts = computed(() => totals.value?.active ?? 0)
const criticalCount = computed(() => totals.value?.critical ?? 0)
const warningCount = computed(() => totals.value?.warning ?? 0)
const mutedCount = computed(() => totals.value?.muted ?? 0)
</script>

<template>
  <div>
    <!-- Counter cards -->
    <div class="mb-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
      <CounterCard :title="t('alerts.total_alerts')" :counter="totalAlerts" :loading="isLoading" />
      <CounterCard
        title="Critical"
        :counter="criticalCount"
        :loading="isLoading"
        color-classes="text-red-600 dark:text-red-400"
      />
      <CounterCard
        title="Warning"
        :counter="warningCount"
        :loading="isLoading"
        color-classes="text-orange-600 dark:text-orange-400"
      />
      <CounterCard :title="t('alerts.muted')" :counter="mutedCount" :loading="isLoading" />
    </div>

    <!-- Alerts table -->
    <AlertsTable />
  </div>
</template>
