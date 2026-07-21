<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import CounterCard from '../common/CounterCard.vue'
import BadgeLink from '../common/BadgeLink.vue'
import { faTriangleExclamation } from '@fortawesome/free-solid-svg-icons'
import { useAlertsTotals } from '@/queries/alerts/alertsTotals.ts'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { abbreviateNumber } from '@/lib/common/index.ts'

const { locale } = useI18n()
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
    :to="{ name: 'alerts' }"
  >
    <div class="mt-5 flex flex-wrap justify-center gap-2">
      <BadgeLink
        v-if="criticalCount > 0"
        :to="{ name: 'alerts', query: { severity: 'critical' } }"
        kind="rose"
        :aria-label="$t('alerts.show_critical_alerts')"
      >
        {{
          $t(
            'alerts.count_critical',
            {
              count: abbreviateNumber(criticalCount, locale),
            },
            criticalCount,
          )
        }}
      </BadgeLink>
      <BadgeLink
        v-if="warningCount > 0"
        :to="{ name: 'alerts', query: { severity: 'warning' } }"
        kind="amber"
        :aria-label="$t('alerts.show_warning_alerts')"
      >
        {{
          $t(
            'alerts.count_warning',
            {
              count: abbreviateNumber(warningCount, locale),
            },
            warningCount,
          )
        }}
      </BadgeLink>
      <BadgeLink
        v-if="mutedCount > 0"
        :to="{ name: 'alerts', query: { status: 'suppressed' } }"
        kind="gray"
        :aria-label="$t('alerts.show_muted_alerts')"
      >
        {{
          $t(
            'alerts.count_muted',
            {
              count: abbreviateNumber(mutedCount, locale),
            },
            mutedCount,
          )
        }}
      </BadgeLink>
    </div>
  </CounterCard>
</template>
