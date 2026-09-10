<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  A ranked list where each row carries a bar sized against the busiest row.
  Three widgets share it: alert severities, noisiest alert rules, most installed
  applications. Like AddonActivationsCard it is divs, not a chart library.

  Rows are optionally links; the `leading` slot takes an application logo.
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NeEmptyState } from '@nethesis/vue-components'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import { faChartSimple } from '@fortawesome/free-solid-svg-icons'
import { abbreviateNumber } from '@/lib/common'

export interface RankedBarRow {
  key: string
  label: string
  count: number
  to?: RouteLocationRaw
  /** Tailwind background classes for the bar; both themes must be given. */
  barClasses?: string
}

const {
  rows,
  emptyTitle,
  defaultBarClasses = 'bg-indigo-600 dark:bg-indigo-400',
} = defineProps<{
  rows: RankedBarRow[]
  emptyTitle: string
  defaultBarClasses?: string
}>()

const { locale } = useI18n()

// A row with a single occurrence still deserves a visible bar, so the scale
// starts above zero — the same reasoning as the add-on activations chart.
const MIN_BAR_PERCENTAGE = 6

const scaledRows = computed(() => {
  const busiest = Math.max(...rows.map((row) => row.count), 0)

  return rows.map((row) => ({
    ...row,
    formattedCount: abbreviateNumber(row.count, locale.value),
    width: busiest
      ? `${MIN_BAR_PERCENTAGE + (row.count / busiest) * (100 - MIN_BAR_PERCENTAGE)}%`
      : '0%',
  }))
})
</script>

<template>
  <NeEmptyState v-if="rows.length === 0" :title="emptyTitle" :icon="faChartSimple" />
  <ul v-else class="flex flex-col gap-3">
    <li v-for="row in scaledRows" :key="row.key">
      <component
        :is="row.to ? RouterLink : 'div'"
        :to="row.to"
        class="group flex flex-col gap-1"
        :class="row.to ? 'cursor-pointer' : undefined"
      >
        <div class="flex items-center justify-between gap-3">
          <span
            class="flex min-w-0 items-center gap-2 text-sm"
            :class="row.to ? 'group-hover:underline' : undefined"
          >
            <slot name="leading" :row="row" />
            <span class="truncate">{{ row.label }}</span>
          </span>
          <span class="text-secondary-neutral shrink-0 text-sm tabular-nums">
            {{ row.formattedCount }}
          </span>
        </div>
        <div class="h-2 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
          <div
            class="h-full rounded-full"
            :class="row.barClasses ?? defaultBarClasses"
            :style="{ width: row.width }"
          />
        </div>
      </component>
    </li>
  </ul>
</template>
