<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  One bar per add-on type, split by the state its grants are in. The report
  endpoint only knows about add-ons somebody has, so the catalog is merged in
  to give the ones nobody has bought a row of their own — on this card that
  absence is the finding.

  Segment colours come from ADDON_STATUS_STYLE, the same table the status icons
  read, so green means active here and everywhere else.
-->

<script setup lang="ts">
import { computed } from 'vue'
import { getStatusSegments, type AddonReportByType } from '@/lib/addons/addonsReport'
import { ADDON_STATUS_STYLE } from '@/lib/addons/systemAddons'
import type { Addon } from '@/lib/addons/addons'
import ReportCard from './ReportCard.vue'

const { byAddon, catalog, loading } = defineProps<{
  byAddon: AddonReportByType[]
  // the whole catalog, so add-ons with no grants at all still get a row;
  // empty when it could not be loaded, which only costs those rows
  catalog: Addon[]
  loading: boolean
}>()

const rows = computed(() => {
  const granted = new Set(byAddon.map((row) => row.entitlement))
  // the endpoint already orders these by total, busiest first
  const ungranted: AddonReportByType[] = catalog
    .filter((addon) => !granted.has(addon.id))
    .map((addon) => ({
      entitlement: addon.id,
      display_name: addon.display_name,
      active: 0,
      expired: 0,
      revoked: 0,
      pending: 0,
      suspended: 0,
      total: 0,
    }))
    .sort((a, b) => a.display_name.localeCompare(b.display_name))

  return [...byAddon, ...ungranted].map((row) => ({
    ...row,
    segments: getStatusSegments(row),
  }))
})
</script>

<template>
  <ReportCard :title="$t('addons.by_addon')" :loading="loading">
    <div class="flex flex-col gap-5">
      <div v-for="row in rows" :key="row.entitlement" class="flex flex-col gap-2">
        <div class="flex items-baseline justify-between gap-4">
          <span class="font-medium text-gray-900 dark:text-gray-100">{{ row.display_name }}</span>
          <span
            :class="[
              'shrink-0 font-medium',
              row.total ? 'text-gray-900 dark:text-gray-100' : 'text-tertiary-neutral',
            ]"
          >
            {{ $t('addons.n_total', { n: row.total }) }}
          </span>
        </div>
        <!-- one track, one flex segment per state present, widths summing to
             100%: the empty track shows through only when nothing is granted -->
        <div
          class="flex h-2.5 w-full overflow-hidden rounded-full bg-gray-300 dark:bg-gray-600"
          role="img"
          :aria-label="row.display_name"
        >
          <div
            v-for="segment in row.segments"
            :key="segment.status"
            :class="ADDON_STATUS_STYLE[segment.status].bar"
            :style="{ width: `${segment.share}%` }"
            :title="`${segment.count} ${$t(`addons.status_${segment.status}`)}`"
          />
        </div>
        <div v-if="row.segments.length" class="flex flex-wrap items-center gap-x-4 gap-y-1">
          <span
            v-for="segment in row.segments"
            :key="segment.status"
            class="text-tertiary-neutral flex items-center gap-2 text-sm font-medium"
          >
            <span
              :class="['size-3 shrink-0 rounded-full', ADDON_STATUS_STYLE[segment.status].bar]"
              aria-hidden="true"
            />
            {{ segment.count }} {{ $t(`addons.status_${segment.status}`) }}
          </span>
        </div>
        <span v-else class="text-tertiary-neutral text-sm font-medium">
          {{ $t('addons.no_systems_with_addon') }}
        </span>
      </div>
    </div>
  </ReportCard>
</template>
