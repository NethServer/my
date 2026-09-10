<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Alert volume split by severity, with the two reliability numbers underneath.
  Shares its query with TopAlertsCard: one /alerts/stats call feeds both.
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatSeconds } from '@/lib/dateTime'
import { useAlertsStats } from '@/queries/dashboard/alertsStats'
import WidgetCard from './WidgetCard.vue'
import RankedBarList, { type RankedBarRow } from './RankedBarList.vue'

const { t } = useI18n()
const { state } = useAlertsStats()

const stats = computed(() => state.value?.data)
const isLoading = computed(() => state.value?.status === 'pending')

// Fixed order and fixed colours: severity is a scale, so it must not reorder
// itself as the counts move.
const SEVERITIES = [
  { key: 'critical', barClasses: 'bg-rose-600 dark:bg-rose-400' },
  { key: 'warning', barClasses: 'bg-amber-500 dark:bg-amber-400' },
  { key: 'info', barClasses: 'bg-sky-600 dark:bg-sky-400' },
] as const

const rows = computed<RankedBarRow[]>(() =>
  SEVERITIES.map((severity) => ({
    key: severity.key,
    label: t(`dashboard.severity_${severity.key}`),
    count: stats.value?.by_severity?.[severity.key] ?? 0,
    to: { name: 'alerts', query: { severity: severity.key } },
    barClasses: severity.barClasses,
  })),
)

const mttr = computed(() =>
  stats.value?.mttr_seconds ? formatSeconds(stats.value.mttr_seconds, t) : undefined,
)
const mtbf = computed(() =>
  stats.value?.mtbf_seconds ? formatSeconds(stats.value.mtbf_seconds, t) : undefined,
)
</script>

<template>
  <WidgetCard :title="$t('dashboard.widget_alerts_severity')" :loading="isLoading">
    <RankedBarList :rows="rows" :empty-title="$t('dashboard.no_firing_alerts')" />
    <dl v-if="mttr || mtbf" class="mt-5 flex flex-wrap gap-x-8 gap-y-2">
      <div v-if="mttr">
        <dt class="text-tertiary-neutral text-xs uppercase">{{ $t('dashboard.mttr') }}</dt>
        <dd class="text-sm">{{ mttr }}</dd>
      </div>
      <div v-if="mtbf">
        <dt class="text-tertiary-neutral text-xs uppercase">{{ $t('dashboard.mtbf') }}</dt>
        <dd class="text-sm">{{ mtbf }}</dd>
      </div>
    </dl>
  </WidgetCard>
</template>
