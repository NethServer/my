<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  How many licenses of an add-on are assigned out of those bought: green while
  some are free, red once they are all taken, so the reader sees at a glance
  whether another instance can still get one.
-->

<script setup lang="ts">
import { faCircleExclamation } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { computed } from 'vue'

const { used, total } = defineProps<{
  used: number
  total: number
}>()

const isFull = computed(() => total > 0 && used >= total)

const percent = computed(() => (total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0))
</script>

<template>
  <div class="flex min-w-32 flex-col items-end gap-1">
    <div class="flex items-center gap-2">
      <FontAwesomeIcon
        v-if="isFull"
        :icon="faCircleExclamation"
        class="size-4 text-rose-700 dark:text-rose-500"
        aria-hidden="true"
      />
      <span
        :class="[
          'font-medium',
          isFull ? 'text-rose-700 dark:text-rose-500' : 'text-gray-900 dark:text-gray-100',
        ]"
      >
        {{ used }}/{{ total }}
      </span>
    </div>
    <div
      class="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"
      role="progressbar"
      :aria-valuenow="used"
      aria-valuemin="0"
      :aria-valuemax="total"
      :aria-label="$t('addons.seats_used_of', { used, total })"
    >
      <div
        :class="[
          'h-full rounded-full',
          isFull ? 'bg-rose-700 dark:bg-rose-500' : 'bg-green-700 dark:bg-green-500',
        ]"
        :style="{ width: `${percent}%` }"
      />
    </div>
    <span v-if="isFull" class="text-xs text-rose-700 dark:text-rose-500">
      {{ $t('addons.licenses_all_assigned') }}
    </span>
  </div>
</template>
