<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  One add-on of a system, summarised over the places it applies to: a line per
  state any of them is in. A NethServer module counts its application
  instances; a NethSecurity service covers the whole firewall, so it states its
  state plainly rather than claiming to be "active on 1 application".
-->

<script setup lang="ts">
import { faArrowRightLong } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeButton, NeCard } from '@nethesis/vue-components'
import { computed } from 'vue'
import ApplicationLogo from '@/components/applications/ApplicationLogo.vue'
import SystemLogo from '@/components/systems/SystemLogo.vue'
import { ADDON_ROW_STATUSES, type AddonRowStatus } from '@/lib/addons/systemAddons'
import type { Addon } from '@/lib/addons/addons'
import AddonStatusIcon from './AddonStatusIcon.vue'

const { addon, applicationId, counts, scoped } = defineProps<{
  addon: Addon
  // the application these rows belong to, '' for a system-wide service
  applicationId: string
  counts: Record<AddonRowStatus, number>
  // false for a NethSecurity service: one place, so no counting
  scoped: boolean
}>()

const emit = defineEmits<{ details: [] }>()

const statusLines = computed(() =>
  ADDON_ROW_STATUSES.filter((status) => counts[status] > 0).map((status) => ({
    status,
    count: counts[status],
  })),
)
</script>

<template>
  <NeCard>
    <div class="flex h-full flex-col gap-5">
      <!-- logo + name + description -->
      <div class="flex flex-col gap-2">
        <div class="flex items-center gap-3">
          <ApplicationLogo v-if="applicationId" :app="applicationId" />
          <SystemLogo v-else system="nsec" />
          <p class="font-medium text-gray-900 dark:text-gray-100">{{ addon.display_name }}</p>
        </div>
        <p v-if="addon.description" class="text-tertiary-neutral">{{ addon.description }}</p>
      </div>
      <!-- one line per state the add-on is in somewhere on this system -->
      <div class="flex flex-col gap-2">
        <AddonStatusIcon
          v-for="line in statusLines"
          :key="line.status"
          :status="line.status"
          :count="scoped ? line.count : undefined"
          class="pb-1 text-sm font-medium"
        />
      </div>
      <!-- pushed to the bottom so cards of differing height line their buttons up -->
      <div class="mt-auto flex justify-end">
        <NeButton kind="tertiary" @click="emit('details')">
          {{ $t('addons.details') }}
          <template #suffix>
            <FontAwesomeIcon :icon="faArrowRightLong" class="size-4" aria-hidden="true" />
          </template>
        </NeButton>
      </div>
    </div>
  </NeCard>
</template>
