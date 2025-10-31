<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeBadgeV2, NeCard, NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { useLatestInventory } from '@/queries/systems/latestInventory'
import {
  faCheck,
  faPowerOff,
  faQuestion,
  faTriangleExclamation,
  faXmark,
} from '@fortawesome/free-solid-svg-icons'
import { formatDateTimeNoSeconds, formatUptime } from '@/lib/dateTime'
import { useI18n } from 'vue-i18n'
import { useSystemDetail } from '@/queries/systems/systemDetail'

const { t, locale } = useI18n()
const { state: systemDetail } = useSystemDetail()
const { state: latestInventory } = useLatestInventory()

const getBadgeKind = () => {
  switch (systemDetail.value.data?.status) {
    case 'online':
      return 'green'
    case 'offline':
      return 'amber'
    case 'deleted':
      return 'rose'
    default:
      return 'gray'
  }
}

const getBadgeIcon = () => {
  switch (systemDetail.value.data?.status) {
    case 'online':
      return faCheck
    case 'offline':
      return faTriangleExclamation
    case 'deleted':
      return faXmark
    default:
      return faQuestion
  }
}
</script>

<template>
  <NeCard>
    <div class="mb-4 flex items-center gap-4">
      <FontAwesomeIcon :icon="faPowerOff" class="size-8 shrink-0" aria-hidden="true" />
      <NeHeading tag="h4">
        {{ $t('common.status') }}
      </NeHeading>
      <!-- status badge -->
      <NeBadgeV2 v-if="systemDetail.status === 'success'" :kind="getBadgeKind()">
        <div class="flex items-center gap-1">
          <FontAwesomeIcon :icon="getBadgeIcon()" class="size-4" />
          {{ t(`systems.status_${systemDetail.data?.status}`) }}
        </div>
      </NeBadgeV2>
      <!-- //// kebab? -->
    </div>
    <!-- get latest inventory error notification -->
    <NeInlineNotification
      v-if="latestInventory.status === 'error'"
      kind="error"
      :title="$t('system_detail.cannot_retrieve_latest_inventory')"
      :description="latestInventory.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="latestInventory.status === 'pending'" :lines="6" />
    <div v-else className="divide-y divide-gray-200 dark:divide-gray-700">
      <!-- uptime -->
      <div class="flex justify-between gap-2 py-4">
        <span class="font-medium">
          {{ $t('system_detail.uptime') }}
        </span>
        <span class="text-gray-600 dark:text-gray-300">
          {{
            latestInventory.data?.data?.system_uptime?.seconds
              ? formatUptime(latestInventory.data?.data?.system_uptime?.seconds, $t)
              : '-'
          }}
        </span>
      </div>
      <!-- last inventory -->
      <div class="flex justify-between gap-2 py-4">
        <span class="font-medium">
          {{ $t('system_detail.last_inventory') }}
        </span>
        <span class="text-gray-600 dark:text-gray-300">
          {{
            latestInventory.data?.timestamp
              ? formatDateTimeNoSeconds(new Date(latestInventory.data?.timestamp), locale)
              : '-'
          }}
        </span>
      </div>
      <!-- timezone -->
      <div class="flex justify-between gap-2 py-4">
        <span class="font-medium">
          {{ $t('system_detail.timezone') }}
        </span>
        <span class="text-gray-600 dark:text-gray-300">
          {{ latestInventory.data?.data?.timezone || '-' }}
        </span>
      </div>
    </div>
  </NeCard>
</template>
