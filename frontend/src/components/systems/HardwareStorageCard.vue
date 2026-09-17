<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { byteFormat1024, NeCard, NeHeading, NeProgressBar } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faHardDrive } from '@fortawesome/free-solid-svg-icons'
import { computed } from 'vue'
import { type HardwareInfo, usageBarColor, usagePercentage } from '@/lib/systems/hardware'

const { hardware } = defineProps<{
  hardware: HardwareInfo
}>()

const mountpoints = computed(() => {
  const mountpointsByPath = hardware.mountpoints

  if (!mountpointsByPath) {
    return []
  }
  return Object.entries(mountpointsByPath)
    .map(([path, mountpoint]) => ({
      path,
      availableBytes: mountpoint.available_bytes,
      totalBytes: mountpoint.total_bytes,
      percentage: usagePercentage(mountpoint.used_bytes, mountpoint.total_bytes),
    }))
    .sort((a, b) => a.path.localeCompare(b.path))
})
</script>

<template>
  <NeCard>
    <div class="mb-4 flex items-center gap-4">
      <FontAwesomeIcon :icon="faHardDrive" class="size-5 shrink-0" aria-hidden="true" />
      <NeHeading tag="h6">
        {{ $t('system_detail.storage').toUpperCase() }}
      </NeHeading>
    </div>
    <div class="divide-y divide-gray-200 dark:divide-gray-700">
      <div v-for="mountpoint in mountpoints" :key="mountpoint.path" class="space-y-2 py-4">
        <div class="flex items-start justify-between gap-4">
          <span class="font-medium break-all">
            {{ mountpoint.path }}
          </span>
          <span class="text-tertiary-neutral dark:text-tertiary-neutral text-end">
            {{
              $t('system_detail.storage_free_of_total', {
                free: byteFormat1024(mountpoint.availableBytes),
                total: byteFormat1024(mountpoint.totalBytes),
              })
            }}
          </span>
        </div>
        <NeProgressBar
          :progress="mountpoint.percentage"
          :color="usageBarColor(mountpoint.percentage)"
          size="sm"
        />
      </div>
    </div>
  </NeCard>
</template>
