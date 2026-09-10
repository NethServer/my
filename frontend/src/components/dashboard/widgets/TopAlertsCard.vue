<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The alert rules that fired most often. Shares the /alerts/stats query with
  AlertsSeverityCard, so a board carrying both still makes one request.
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useAlertsStats } from '@/queries/dashboard/alertsStats'
import WidgetCard from './WidgetCard.vue'
import RankedBarList, { type RankedBarRow } from './RankedBarList.vue'

const { state } = useAlertsStats()

const isLoading = computed(() => state.value?.status === 'pending')

const rows = computed<RankedBarRow[]>(() =>
  (state.value?.data?.top_alertnames ?? [])
    .filter((entry) => !!entry.alertname)
    .map((entry) => ({
      key: entry.alertname ?? '',
      label: entry.alertname ?? '',
      count: entry.count,
      to: { name: 'alerts', query: { alertname: entry.alertname } },
    })),
)
</script>

<template>
  <WidgetCard :title="$t('dashboard.widget_alerts_top_names')" :loading="isLoading">
    <RankedBarList :rows="rows" :empty-title="$t('dashboard.no_firing_alerts')" />
  </WidgetCard>
</template>
