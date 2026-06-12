<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import CounterCard from '../CounterCard.vue'
import { NeBadgeV2 } from '@nethesis/vue-components'
import { faTriangleExclamation } from '@fortawesome/free-solid-svg-icons'
import { useAlertsTotals } from '@/queries/alerts/alertsTotals.ts'
import { computed } from 'vue'

const { state: totalsState } = useAlertsTotals()

const totals = computed(() => totalsState.value?.data)
const isLoading = computed(() => totalsState.value?.status === 'pending')

const totalCount = computed(() => totals.value?.active ?? 0)
const criticalCount = computed(() => totals.value?.critical ?? 0)
const warningCount = computed(() => totals.value?.warning ?? 0)
const mutedCount = computed(() => totals.value?.muted ?? 0)
</script>

<template>
  <CounterCard
    :title="$t('alerts.total_alerts')"
    :counter="totalCount"
    :icon="faTriangleExclamation"
    :loading="isLoading"
    title-route-name="alerts"
  >
    <div class="mt-5 flex flex-wrap justify-center gap-2">
      <NeBadgeV2 v-if="criticalCount > 0" kind="rose">
        {{ $t('alerts.count_critical', { count: criticalCount }) }}
      </NeBadgeV2>
      <NeBadgeV2 v-if="warningCount > 0" kind="amber">
        {{ $t('alerts.count_warning', { count: warningCount }) }}
      </NeBadgeV2>
      <NeBadgeV2 v-if="mutedCount > 0" kind="gray">
        {{ $t('alerts.count_muted', { count: mutedCount }) }}
      </NeBadgeV2>
    </div>
  </CounterCard>
</template>
