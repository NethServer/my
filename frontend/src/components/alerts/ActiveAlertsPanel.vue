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
import { useLoginStore } from '@/stores/login'

const MIN_ESTIMATED_COUNT = 50

const { t } = useI18n()
const loginStore = useLoginStore()
const { state: totalsState } = useAlertsTotals()

const totals = computed(() => totalsState.value?.data)
const isLoading = computed(() => totalsState.value?.status === 'pending')

const totalCount = computed(() => totals.value?.active ?? 0)
const criticalCount = computed(() => totals.value?.critical ?? 0)
const warningCount = computed(() => totals.value?.warning ?? 0)
const infoCount = computed(() => totals.value?.info ?? 0)
const mutedCount = computed(() => totals.value?.muted ?? 0)
</script>

<template>
  <div>
    <!-- Counter cards -->
    <div class="mb-10 grid gap-6 sm:grid-cols-6 xl:grid-cols-5">
      <CounterCard
        :title="t('alerts.total_alerts')"
        :counter="totalCount"
        :loading="isLoading"
        colorClasses="text-secondary-neutral dark:text-secondary-neutral"
        :is-estimated="loginStore.isOwner && totalCount > MIN_ESTIMATED_COUNT"
        class="sm:col-span-3 xl:col-span-1"
      />
      <CounterCard
        :title="t('alerts.muted')"
        :counter="mutedCount"
        :loading="isLoading"
        color-classes="text-secondary-neutral dark:text-secondary-neutral"
        :is-estimated="loginStore.isOwner && mutedCount > MIN_ESTIMATED_COUNT"
        class="sm:col-span-3 xl:col-span-1"
      />
      <CounterCard
        title="Critical"
        :counter="criticalCount"
        :loading="isLoading"
        color-classes="text-rose-600 dark:text-rose-400"
        :is-estimated="loginStore.isOwner && criticalCount > MIN_ESTIMATED_COUNT"
        class="sm:col-span-2 xl:col-span-1"
      />
      <CounterCard
        title="Warning"
        :counter="warningCount"
        :loading="isLoading"
        color-classes="text-amber-600 dark:text-amber-400"
        :is-estimated="loginStore.isOwner && warningCount > MIN_ESTIMATED_COUNT"
        class="sm:col-span-2 xl:col-span-1"
      />
      <CounterCard
        title="Info"
        :counter="infoCount"
        :loading="isLoading"
        color-classes="text-blue-600 dark:text-blue-400"
        :is-estimated="loginStore.isOwner && infoCount > MIN_ESTIMATED_COUNT"
        class="sm:col-span-2 xl:col-span-1"
      />
    </div>

    <!-- Alerts table -->
    <AlertsTable />
  </div>
</template>
