<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeHeading } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faMicrochip } from '@fortawesome/free-solid-svg-icons'
import type { HardwareInfo } from '@/lib/systems/hardware'
import DataItem from '../common/DataItem.vue'

const { hardware } = defineProps<{
  hardware: HardwareInfo
}>()
</script>

<template>
  <NeCard>
    <div class="mb-4 flex items-center gap-4">
      <FontAwesomeIcon :icon="faMicrochip" class="size-5 shrink-0" aria-hidden="true" />
      <NeHeading tag="h6">
        {{ $t('system_detail.cpu_and_bios').toUpperCase() }}
      </NeHeading>
    </div>
    <div class="divide-y divide-gray-200 dark:divide-gray-700">
      <!-- model -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.model') }}
        </template>
        <template #data>
          {{ hardware.processors?.model || '-' }}
        </template>
      </DataItem>
      <!-- processors count -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.processors_count') }}
        </template>
        <template #data>
          {{ hardware.processors?.count ?? '-' }}
        </template>
      </DataItem>
      <!-- architecture -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.architecture') }}
        </template>
        <template #data>
          {{ hardware.processors?.architecture || '-' }}
        </template>
      </DataItem>
      <!-- bios version -->
      <DataItem v-if="hardware.bios">
        <template #label>
          {{ $t('system_detail.bios_version') }}
        </template>
        <template #data>
          {{ hardware.bios.version || '-' }}
        </template>
      </DataItem>
      <!-- bios vendor -->
      <DataItem v-if="hardware.bios">
        <template #label>
          {{ $t('system_detail.bios_vendor') }}
        </template>
        <template #data>
          {{ hardware.bios.vendor || '-' }}
        </template>
      </DataItem>
    </div>
  </NeCard>
</template>
