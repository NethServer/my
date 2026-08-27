<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Grants created per month over the last year. A column chart this plain is a
  dozen divs whose heights are a share of the busiest month, so it needs no
  charting library — and it inherits the app's colours and dark mode for free.
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { fillTrendMonths, type AddonReportTrendRow } from '@/lib/addons/addonsReport'
import ReportCard from './ReportCard.vue'

const { trend, loading } = defineProps<{
  trend: AddonReportTrendRow[]
  loading: boolean
}>()

const { locale } = useI18n()

// A month with a single activation still deserves a visible bar, so the scale
// starts here rather than at zero; the busiest month stops short of the top to
// leave its count somewhere to sit.
const MIN_BAR_PERCENTAGE = 4
const MAX_BAR_PERCENTAGE = 88

const months = computed(() => {
  const rows = fillTrendMonths(trend, new Date())
  const busiest = Math.max(...rows.map((row) => row.activations), 0)

  return rows.map((row) => {
    const [year, month] = row.month.split('-').map(Number)
    const date = new Date(year, month - 1, 1)

    return {
      ...row,
      label: date.toLocaleDateString(locale.value, { month: 'short' }),
      // the axis shows the month alone, so the tooltip carries the year
      title: date.toLocaleDateString(locale.value, { month: 'long', year: 'numeric' }),
      height: busiest
        ? `${MIN_BAR_PERCENTAGE + (row.activations / busiest) * (MAX_BAR_PERCENTAGE - MIN_BAR_PERCENTAGE)}%`
        : '0%',
    }
  })
})

const hasActivations = computed(() => months.value.some((month) => month.activations > 0))
</script>

<template>
  <ReportCard :title="$t('addons.activations_last_12_months')" :loading="loading">
    <div class="flex h-64 items-stretch gap-1 sm:gap-2">
      <div
        v-for="month in months"
        :key="month.month"
        class="flex flex-1 flex-col items-center gap-2"
        :title="`${month.title}: ${month.activations}`"
      >
        <!-- the count rides directly on top of its own bar rather than on a
             shared line, so a short bar is not read against a distant number -->
        <div class="flex w-full flex-1 flex-col items-center justify-end gap-1">
          <span class="text-secondary-neutral text-sm">{{ month.activations || '' }}</span>
          <div
            v-if="month.activations"
            class="w-full max-w-16 rounded-t-md bg-indigo-600 dark:bg-indigo-400"
            :style="{ height: month.height }"
          />
        </div>
        <span class="text-secondary-neutral text-sm capitalize">{{ month.label }}</span>
      </div>
    </div>
    <!-- a fleet can hold plenty of add-ons and still have activated none this
         year: a bare grid of month labels would read as a bug -->
    <p v-if="!hasActivations" class="text-tertiary-neutral mt-4 text-center text-sm">
      {{ $t('addons.no_activations') }}
    </p>
  </ReportCard>
</template>
