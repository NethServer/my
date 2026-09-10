<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Renewals coming due at 30, 60 and 90 days. The three windows are cumulative
  — 30d ⊆ 60d ⊆ 90d — which is why they are drawn as nested bars against the
  active total and labelled as such: three side-by-side bars would read as
  three disjoint groups and overcount the work ahead.

  Reuses the add-ons page's own report query, so a user who has been there pays
  for no second request.
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useAddonReport } from '@/queries/addons/addonsReport'
import WidgetCard from './WidgetCard.vue'
import RankedBarList, { type RankedBarRow } from './RankedBarList.vue'

const { state } = useAddonReport()

const isLoading = computed(() => state.value?.status === 'pending')
const totals = computed(() => state.value?.data?.totals)

const WINDOWS = [
  { key: '30d', field: 'expiring_in_30d', barClasses: 'bg-rose-600 dark:bg-rose-400' },
  { key: '60d', field: 'expiring_in_60d', barClasses: 'bg-amber-500 dark:bg-amber-400' },
  { key: '90d', field: 'expiring_in_90d', barClasses: 'bg-gray-500 dark:bg-gray-400' },
] as const

const rows = computed<RankedBarRow[]>(() =>
  WINDOWS.map((window) => ({
    key: window.key,
    label: window.key,
    count: totals.value?.[window.field] ?? 0,
    barClasses: window.barClasses,
  })),
)
</script>

<template>
  <WidgetCard
    :title="$t('dashboard.widget_addons_expiry_windows')"
    :loading="isLoading"
    :skeleton-lines="4"
  >
    <RankedBarList :rows="rows" :empty-title="$t('dashboard.no_expiring_addons')" />
    <p class="text-tertiary-neutral mt-4 text-xs">{{ $t('dashboard.cumulative_note') }}</p>
  </WidgetCard>
</template>
