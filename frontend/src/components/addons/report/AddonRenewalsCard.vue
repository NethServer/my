<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  How often the fleet's grants have been renewed. Four buckets over the same
  denominator — every grant in view — so the bars are comparable down the card.
-->

<script setup lang="ts">
import { NeProgressBar } from '@nethesis/vue-components'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AddonReportRenewals } from '@/lib/addons/addonsReport'
import ReportCard from './ReportCard.vue'

const { renewals, total, loading } = defineProps<{
  renewals: AddonReportRenewals
  // every grant in view, renewed or not: the denominator of all four rows
  total: number
  loading: boolean
}>()

const { t } = useI18n()

const rows = computed(() =>
  (
    [
      ['never', t('addons.never_renewed')],
      ['once', t('addons.renewed_once')],
      ['twice', t('addons.renewed_twice')],
      ['three_plus', t('addons.renewed_three_plus')],
    ] as const
  ).map(([bucket, label]) => ({
    bucket,
    label,
    count: renewals[bucket],
    progress: total ? (renewals[bucket] / total) * 100 : 0,
  })),
)
</script>

<template>
  <ReportCard :title="$t('addons.renewal_distribution')" :loading="loading">
    <div class="flex flex-col gap-6">
      <div v-for="row in rows" :key="row.bucket" class="flex flex-col gap-2">
        <div class="flex items-baseline justify-between gap-4">
          <span class="font-medium text-gray-900 dark:text-gray-100">{{ row.label }}</span>
          <span class="font-medium text-gray-900 dark:text-gray-100">
            {{ row.count }}/{{ total }}
          </span>
        </div>
        <NeProgressBar :progress="row.progress" size="md" color="indigo" />
      </div>
    </div>
  </ReportCard>
</template>
