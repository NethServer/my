<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  How many licenses of an add-on are assigned out of those bought. A full bar
  stays the colour of any other: running out is not an error, and the button
  that can no longer assign one says why.
-->

<script setup lang="ts">
import { NeProgressBar } from '@nethesis/vue-components'
import { computed } from 'vue'

const { used, total } = defineProps<{
  used: number
  total: number
}>()

const percent = computed(() => (total > 0 ? Math.min(100, Math.round((used / total) * 100)) : 0))
</script>

<template>
  <div class="flex flex-col gap-2">
    <div class="flex items-center justify-between gap-4">
      <span class="text-tertiary-neutral font-medium">{{ $t('addons.seats') }}</span>
      <span class="text-secondary-neutral">{{ used }}/{{ total }}</span>
    </div>
    <div
      role="progressbar"
      :aria-valuenow="used"
      aria-valuemin="0"
      :aria-valuemax="total"
      :aria-label="$t('addons.seats_used_of', { used, total })"
    >
      <NeProgressBar
        :progress="percent"
        size="sm"
        color="custom"
        custom-color-classes="bg-primary-active"
      />
    </div>
  </div>
</template>
