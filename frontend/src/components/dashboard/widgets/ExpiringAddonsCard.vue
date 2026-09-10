<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Add-on grants expiring inside the next 30 days. The one widget on the board
  that is a to-do list rather than a measurement, so an empty state here is good
  news and says so.
-->

<script setup lang="ts">
import { computed } from 'vue'
import { NeBadgeV2, NeEmptyState, type NeBadgeV2Kind } from '@nethesis/vue-components'
import { RouterLink } from 'vue-router'
import { faCircleCheck } from '@fortawesome/free-solid-svg-icons'
import { useExpiringAddons } from '@/queries/dashboard/expiringAddons'
import WidgetCard from './WidgetCard.vue'

const { state } = useExpiringAddons()

const isLoading = computed(() => state.value?.status === 'pending')

// A week is the point where a renewal stops being a note and starts being an
// interruption, so that is where the badge changes colour.
const URGENT_DAYS = 7

const daysUntil = (isoDate?: string) => {
  if (!isoDate) {
    return undefined
  }
  const millis = new Date(isoDate).getTime() - Date.now()
  return Math.max(0, Math.ceil(millis / (24 * 60 * 60 * 1000)))
}

const grants = computed(() =>
  (state.value?.data?.grants ?? []).map((grant) => {
    const days = daysUntil(grant.expires_at)
    return {
      id: grant.id,
      label: grant.display_name || grant.entitlement,
      system: grant.system_name ?? grant.organization_name ?? '',
      days,
      badgeKind: (days !== undefined && days <= URGENT_DAYS ? 'rose' : 'amber') as NeBadgeV2Kind,
    }
  }),
)
</script>

<template>
  <WidgetCard :title="$t('dashboard.widget_addons_expiring')" :loading="isLoading">
    <NeEmptyState
      v-if="grants.length === 0"
      :title="$t('dashboard.no_expiring_addons')"
      :icon="faCircleCheck"
    />
    <ul v-else class="flex flex-col gap-3">
      <li v-for="grant in grants" :key="grant.id">
        <RouterLink :to="{ name: 'addons' }" class="group flex items-center justify-between gap-3">
          <span class="min-w-0">
            <span class="block truncate text-sm group-hover:underline">{{ grant.label }}</span>
            <span class="text-tertiary-neutral block truncate text-xs">{{ grant.system }}</span>
          </span>
          <NeBadgeV2 v-if="grant.days !== undefined" :kind="grant.badgeKind" class="shrink-0">
            {{ $t('dashboard.expiring_in_days', { count: grant.days }, grant.days) }}
          </NeBadgeV2>
        </RouterLink>
      </li>
    </ul>
  </WidgetCard>
</template>
