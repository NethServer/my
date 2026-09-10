<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The applications installed on most systems. The backend already sorts by
  count, so page_size=5 is the top five with no client-side ranking.
-->

<script setup lang="ts">
import { computed } from 'vue'
import ApplicationLogo from '@/components/applications/ApplicationLogo.vue'
import { useTopApplications } from '@/queries/dashboard/topApplications'
import WidgetCard from './WidgetCard.vue'
import RankedBarList, { type RankedBarRow } from './RankedBarList.vue'

const { state } = useTopApplications()

const isLoading = computed(() => state.value?.status === 'pending')

const rows = computed<RankedBarRow[]>(() =>
  (state.value?.data?.by_type ?? []).map((entry) => ({
    key: entry.instance_of,
    label: entry.name || entry.instance_of,
    count: entry.count,
    to: { name: 'applications' },
  })),
)
</script>

<template>
  <WidgetCard :title="$t('dashboard.widget_applications_top')" :loading="isLoading">
    <RankedBarList :rows="rows" :empty-title="$t('dashboard.no_applications')">
      <template #leading="{ row }">
        <ApplicationLogo :app="row.key" class="size-5 shrink-0" />
      </template>
    </RankedBarList>
  </WidgetCard>
</template>
