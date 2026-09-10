<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  One trend widget serves every /trend endpoint: they share a contract, so the
  five trend widgets on the board are this component with a different resource.

  The sparkline is an inline SVG rather than a charting library, for the reason
  AddonActivationsCard states: a polyline over a normalised series needs no
  dependency, and `stroke-current` gives it the app's colours and dark mode.
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NeBadgeV2, type NeBadgeV2Kind } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowTrendDown, faArrowTrendUp, faMinus } from '@fortawesome/free-solid-svg-icons'
import { abbreviateNumber } from '@/lib/common'
import { buildSparkline, type ResourceTrend } from '@/lib/dashboard/trends'
import WidgetCard from './WidgetCard.vue'

const {
  title,
  trend = undefined,
  loading = false,
} = defineProps<{
  title: string
  trend?: ResourceTrend
  loading?: boolean
}>()

const { locale } = useI18n()

const sparkline = computed(() => buildSparkline(trend?.data_points ?? []))
const hasSeries = computed(() => (trend?.data_points?.length ?? 0) > 0)

const total = computed(() => abbreviateNumber(trend?.current_total ?? 0, locale.value))

// A flat period is neither good nor bad news, so it gets the neutral badge; up
// and down are coloured only because the reader is comparing them to each other,
// not because more is better — a rising alert count is not an improvement.
const badgeKind = computed<NeBadgeV2Kind>(() => {
  switch (trend?.trend) {
    case 'up':
      return 'blue'
    case 'down':
      return 'gray'
    default:
      return 'gray'
  }
})

const badgeIcon = computed(() => {
  switch (trend?.trend) {
    case 'up':
      return faArrowTrendUp
    case 'down':
      return faArrowTrendDown
    default:
      return faMinus
  }
})

const deltaLabel = computed(() => {
  const percentage = trend?.delta_percentage ?? 0
  const sign = percentage > 0 ? '+' : ''
  return `${sign}${Math.round(percentage * 10) / 10}%`
})
</script>

<template>
  <WidgetCard :title="title" :loading="loading" :skeleton-lines="4">
    <!--
      Natural height, not h-full: the card fills the cell, and content that
      tried to stretch inside it would only overflow, which NeCard clips.
    -->
    <div class="flex flex-col gap-4">
      <div class="flex items-baseline gap-3">
        <span class="text-3xl font-semibold">{{ total }}</span>
        <NeBadgeV2 :kind="badgeKind">
          <FontAwesomeIcon :icon="badgeIcon" class="size-4" aria-hidden="true" />
          {{ deltaLabel }}
        </NeBadgeV2>
      </div>

      <svg
        v-if="hasSeries"
        class="h-16 w-full text-indigo-600 dark:text-indigo-400"
        viewBox="0 0 100 32"
        preserveAspectRatio="none"
        role="img"
        :aria-label="$t('dashboard.trend_last_30_days')"
      >
        <polygon :points="sparkline.area" fill="currentColor" opacity="0.12" />
        <polyline
          :points="sparkline.line"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linejoin="round"
          stroke-linecap="round"
          vector-effect="non-scaling-stroke"
        />
      </svg>
      <p v-else class="text-tertiary-neutral text-sm">{{ $t('dashboard.trend_no_data') }}</p>

      <p class="text-secondary-neutral text-sm">{{ $t('dashboard.trend_last_30_days') }}</p>
    </div>
  </WidgetCard>
</template>
