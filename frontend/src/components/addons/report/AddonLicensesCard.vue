<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The add-ons sold by licenses that are in force: how many of the licenses
  bought are assigned to application instances, and how many add-ons have
  none left — the systems where the next instance needs a higher tier.
-->

<script setup lang="ts">
import { computed } from 'vue'
import type { AddonReportLicenses } from '@/lib/addons/addonsReport'
import ReportCard from './ReportCard.vue'

const { licenses, loading } = defineProps<{
  licenses: AddonReportLicenses
  loading: boolean
}>()

const percent = computed(() =>
  licenses.bought ? Math.min(100, Math.round((licenses.assigned / licenses.bought) * 100)) : 0,
)
</script>

<template>
  <ReportCard :title="$t('addons.licenses_report')" :loading="loading">
    <p v-if="!licenses.addons" class="text-tertiary-neutral">
      {{ $t('addons.no_licensed_addons') }}
    </p>
    <div v-else class="flex flex-col gap-6">
      <div class="flex flex-col gap-2">
        <div class="flex items-baseline justify-between gap-4">
          <span class="font-medium text-gray-900 dark:text-gray-100">
            {{ $t('addons.licenses_assigned') }}
          </span>
          <span class="font-medium text-gray-900 dark:text-gray-100">
            {{ licenses.assigned }}/{{ licenses.bought }}
          </span>
        </div>
        <div class="h-2 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
          <div
            class="h-full rounded-full bg-green-700 dark:bg-green-500"
            :style="{ width: `${percent}%` }"
          />
        </div>
      </div>
      <div class="flex items-baseline justify-between gap-4">
        <span class="text-gray-900 dark:text-gray-100">
          {{ $t('addons.licensed_addons') }}
        </span>
        <span class="font-medium text-gray-900 dark:text-gray-100">{{ licenses.addons }}</span>
      </div>
      <div class="flex items-baseline justify-between gap-4">
        <span class="text-gray-900 dark:text-gray-100">
          {{ $t('addons.licensed_addons_full') }}
        </span>
        <span
          :class="[
            'font-medium',
            licenses.full ? 'text-rose-700 dark:text-rose-500' : 'text-gray-900 dark:text-gray-100',
          ]"
        >
          {{ licenses.full }}
        </span>
      </div>
    </div>
  </ReportCard>
</template>
