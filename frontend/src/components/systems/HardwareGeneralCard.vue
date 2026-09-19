<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeAvatar, NeCard, NeHeading } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faCircleInfo, faServer } from '@fortawesome/free-solid-svg-icons'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { HardwareInfo } from '@/lib/systems/hardware'
import { formatUptime } from '@/lib/dateTime'
import DataItem from '../common/DataItem.vue'
import ClickToCopy from '../common/ClickToCopy.vue'

const { hardware } = defineProps<{
  hardware: HardwareInfo
}>()

const { t } = useI18n()

// Empty on bare metal, otherwise the hypervisor, e.g. 'kvm'
const hypervisor = computed(() => {
  const virtual = hardware.virtual.trim()
  return virtual.toLowerCase() === 'physical' ? '' : virtual
})

const distribution = computed(() => {
  const { name, version } = hardware.distro || {}

  if (!name) {
    return ''
  }
  return version ? `${name} ${version}` : name
})
</script>

<template>
  <NeCard>
    <div class="mb-4 flex items-center gap-4">
      <FontAwesomeIcon :icon="faCircleInfo" class="size-5 shrink-0" aria-hidden="true" />
      <NeHeading tag="h6">
        {{ $t('system_detail.general').toUpperCase() }}
      </NeHeading>
    </div>
    <!-- product name and virtualization -->
    <div class="my-6 flex items-center gap-6">
      <NeAvatar size="2xl" aria-hidden="true">
        <template #placeholder>
          <div
            class="flex size-16 items-center justify-center rounded-full bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-50"
          >
            <FontAwesomeIcon :icon="faServer" class="size-8" />
          </div>
        </template>
      </NeAvatar>
      <div class="flex flex-col gap-1">
        <span class="text-lg font-medium">
          {{ hardware.productName || '-' }}
        </span>
        <span class="text-tertiary-neutral dark:text-tertiary-neutral">
          {{ hypervisor ? $t('system_detail.virtual') : $t('system_detail.physical') }}
          <span v-if="hypervisor">&bull; {{ hypervisor }}</span>
        </span>
      </div>
    </div>
    <div class="divide-y divide-gray-200 dark:divide-gray-700">
      <!-- manufacturer -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.manufacturer') }}
        </template>
        <template #data>
          {{ hardware.manufacturer || '-' }}
        </template>
      </DataItem>
      <!-- distribution -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.distribution') }}
        </template>
        <template #data>
          {{ distribution || '-' }}
        </template>
      </DataItem>
      <!-- kernel version -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.kernel_version') }}
        </template>
        <template #data>
          <span class="break-all">
            {{ hardware.kernelVersion || '-' }}
          </span>
        </template>
      </DataItem>
      <!-- uptime -->
      <DataItem v-if="hardware.uptimeSeconds !== undefined">
        <template #label>
          {{ $t('system_detail.uptime') }}
        </template>
        <template #data>
          {{ formatUptime(hardware.uptimeSeconds, t) }}
        </template>
      </DataItem>
      <!-- timezone -->
      <DataItem v-if="hardware.timezone !== undefined">
        <template #label>
          {{ $t('system_detail.timezone') }}
        </template>
        <template #data>
          {{ hardware.timezone || '-' }}
        </template>
      </DataItem>
      <!-- uuid -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.uuid') }}
        </template>
        <template #data>
          <ClickToCopy v-if="hardware.uuid" :text="hardware.uuid" tooltip-placement="left" />
          <span v-else>-</span>
        </template>
      </DataItem>
    </div>
  </NeCard>
</template>
