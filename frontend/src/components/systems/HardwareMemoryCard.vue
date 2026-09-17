<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { byteFormat1024, NeCard, NeHeading, NeProgressBar } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faMemory } from '@fortawesome/free-solid-svg-icons'
import { computed } from 'vue'
import { type HardwareInfo, usageBarColor, usagePercentage } from '@/lib/systems/hardware'
import type { MemoryUsage } from '@/lib/systems/inventory'

const { hardware } = defineProps<{
  hardware: HardwareInfo
}>()

interface MemoryRow {
  id: string
  label: string
  usedBytes: number
  totalBytes: number
  percentage: number
}

const toRow = (id: string, label: string, usage: MemoryUsage | undefined): MemoryRow => {
  const usedBytes = usage?.used_bytes || 0
  const totalBytes = usage?.total_bytes || usedBytes + (usage?.available_bytes || 0)

  return { id, label, usedBytes, totalBytes, percentage: usagePercentage(usedBytes, totalBytes) }
}

// Swap is hidden when the machine has none, rather than shown as a zeroed bar
const memoryRows = computed(() => {
  const ram = toRow('ram', 'system_detail.ram', hardware.memory?.system)
  const swap = toRow('swap', 'system_detail.swap', hardware.memory?.swap)

  return swap.totalBytes ? [ram, swap] : [ram]
})
</script>

<template>
  <NeCard>
    <div class="mb-4 flex items-center gap-4">
      <FontAwesomeIcon :icon="faMemory" class="size-5 shrink-0" aria-hidden="true" />
      <NeHeading tag="h6">
        {{ $t('system_detail.memory').toUpperCase() }}
      </NeHeading>
    </div>
    <div class="divide-y divide-gray-200 dark:divide-gray-700">
      <div v-for="usage in memoryRows" :key="usage.id" class="space-y-2 py-4">
        <div class="flex items-start justify-between gap-4">
          <span class="font-medium">
            {{ $t(usage.label) }}
          </span>
          <span class="text-tertiary-neutral dark:text-tertiary-neutral text-end">
            {{
              $t('system_detail.memory_used_of_total', {
                used: byteFormat1024(usage.usedBytes),
                total: byteFormat1024(usage.totalBytes),
              })
            }}
          </span>
        </div>
        <NeProgressBar
          :progress="usage.percentage"
          :color="usageBarColor(usage.percentage)"
          size="sm"
        />
      </div>
    </div>
  </NeCard>
</template>
