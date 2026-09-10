<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The most severe alerts firing right now, as an action list. The one widget on
  the board that polls, on the same interval and the same visibility guard as
  the alerts page.
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NeBadgeV2, NeEmptyState, formatRelativeTime } from '@nethesis/vue-components'
import { RouterLink } from 'vue-router'
import { faBell } from '@fortawesome/free-solid-svg-icons'
import { getSeverityBadgeKind } from '@/lib/alerts'
import { useFiringAlerts } from '@/queries/dashboard/firingAlerts'
import WidgetCard from './WidgetCard.vue'

const { locale } = useI18n()
const { state } = useFiringAlerts()

const isLoading = computed(() => state.value?.status === 'pending')

const alerts = computed(() =>
  (state.value?.data?.alerts ?? []).map((alert) => ({
    fingerprint: alert.fingerprint,
    severity: alert.labels.severity,
    alertname: alert.labels.alertname ?? '',
    systemName: alert.labels.system_name ?? alert.labels.system_key ?? '',
    startedAt: formatRelativeTime(new Date(alert.startsAt), locale.value),
  })),
)
</script>

<template>
  <WidgetCard :title="$t('dashboard.widget_alerts_firing_now')" :loading="isLoading">
    <NeEmptyState
      v-if="alerts.length === 0"
      :title="$t('dashboard.no_firing_alerts')"
      :icon="faBell"
    />
    <ul v-else class="flex flex-col gap-3">
      <li v-for="alert in alerts" :key="alert.fingerprint">
        <RouterLink :to="{ name: 'alerts' }" class="group flex items-center justify-between gap-3">
          <span class="flex min-w-0 items-center gap-2">
            <NeBadgeV2 :kind="getSeverityBadgeKind(alert.severity)">
              {{ alert.severity }}
            </NeBadgeV2>
            <span class="min-w-0">
              <span class="block truncate text-sm group-hover:underline">
                {{ alert.alertname }}
              </span>
              <span class="text-tertiary-neutral block truncate text-xs">
                {{ alert.systemName }}
              </span>
            </span>
          </span>
          <span class="text-secondary-neutral shrink-0 text-xs">{{ alert.startedAt }}</span>
        </RouterLink>
      </li>
    </ul>
  </WidgetCard>
</template>
