<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The systems registered most recently. Useful as a "did the onboarding land"
  glance, which a total alone never answers.
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NeBadgeV2, NeEmptyState } from '@nethesis/vue-components'
import { formatDateNoTime } from '@/lib/dateTime'
import { RouterLink } from 'vue-router'
import { faServer } from '@fortawesome/free-solid-svg-icons'
import { useRecentSystems } from '@/queries/dashboard/recentSystems'
import WidgetCard from './WidgetCard.vue'

const { locale } = useI18n()
const { state } = useRecentSystems()

const isLoading = computed(() => state.value?.status === 'pending')

const systems = computed(() =>
  (state.value?.data?.systems ?? []).map((system) => ({
    id: system.id,
    name: system.name,
    type: system.type,
    organization: system.organization?.name ?? '',
    createdAt: system.created_at ? formatDateNoTime(new Date(system.created_at), locale.value) : '',
  })),
)
</script>

<template>
  <WidgetCard :title="$t('dashboard.widget_systems_recent')" :loading="isLoading">
    <NeEmptyState
      v-if="systems.length === 0"
      :title="$t('dashboard.no_recent_systems')"
      :icon="faServer"
    />
    <ul v-else class="flex flex-col gap-3">
      <li v-for="system in systems" :key="system.id">
        <RouterLink
          :to="{ name: 'system_detail', params: { systemId: system.id } }"
          class="group flex items-center justify-between gap-3"
        >
          <span class="flex min-w-0 items-center gap-2">
            <NeBadgeV2 v-if="system.type" kind="gray">{{ system.type }}</NeBadgeV2>
            <span class="min-w-0">
              <span class="block truncate text-sm group-hover:underline">{{ system.name }}</span>
              <span class="text-tertiary-neutral block truncate text-xs">
                {{ system.organization }}
              </span>
            </span>
          </span>
          <span class="text-secondary-neutral shrink-0 text-xs">{{ system.createdAt }}</span>
        </RouterLink>
      </li>
    </ul>
  </WidgetCard>
</template>
