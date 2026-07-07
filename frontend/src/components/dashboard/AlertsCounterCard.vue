<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import CounterCard from '../common/CounterCard.vue'
import { NeBadgeV2 } from '@nethesis/vue-components'
import { faTriangleExclamation } from '@fortawesome/free-solid-svg-icons'
import { useAlertsTotals } from '@/queries/alerts/alertsTotals.ts'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { abbreviateNumber } from '@/lib/common/index.ts'
import { useLoginStore } from '@/stores/login'
import { MIN_ESTIMATED_COUNT } from '@/lib/alerts.ts'

const { locale } = useI18n()
const loginStore = useLoginStore()
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
    :is-estimated="loginStore.isOwner && totalCount > MIN_ESTIMATED_COUNT"
  >
    <div class="mt-5 flex flex-wrap justify-center gap-2">
      <router-link
        v-if="criticalCount > 0"
        :to="{ name: 'alerts', query: { severity: 'critical' } }"
        class="group"
        :aria-label="$t('alerts.show_critical_alerts')"
      >
        <NeBadgeV2 kind="rose" class="group-hover:underline">
          {{
            $t(
              'alerts.count_critical',
              {
                count:
                  (loginStore.isOwner && criticalCount > MIN_ESTIMATED_COUNT ? '~' : '') +
                  abbreviateNumber(criticalCount, locale),
              },
              criticalCount,
            )
          }}
        </NeBadgeV2>
      </router-link>
      <router-link
        v-if="warningCount > 0"
        :to="{ name: 'alerts', query: { severity: 'warning' } }"
        class="group"
        :aria-label="$t('alerts.show_warning_alerts')"
      >
        <NeBadgeV2 kind="amber" class="group-hover:underline">
          {{
            $t(
              'alerts.count_warning',
              {
                count:
                  (loginStore.isOwner && warningCount > MIN_ESTIMATED_COUNT ? '~' : '') +
                  abbreviateNumber(warningCount, locale),
              },
              warningCount,
            )
          }}
        </NeBadgeV2>
      </router-link>
      <router-link
        v-if="mutedCount > 0"
        :to="{ name: 'alerts', query: { status: 'suppressed' } }"
        class="group"
        :aria-label="$t('alerts.show_muted_alerts')"
      >
        <NeBadgeV2 kind="gray" class="group-hover:underline">
          {{
            $t(
              'alerts.count_muted',
              {
                count:
                  (loginStore.isOwner && mutedCount > MIN_ESTIMATED_COUNT ? '~' : '') +
                  abbreviateNumber(mutedCount, locale),
              },
              mutedCount,
            )
          }}
        </NeBadgeV2>
      </router-link>
    </div>
  </CounterCard>
</template>
