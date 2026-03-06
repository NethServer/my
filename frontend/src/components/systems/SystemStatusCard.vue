<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeBadgeV2,
  NeCard,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { useLatestInventory } from '@/queries/systems/latestInventory'
import {
  faCheck,
  faCircleInfo,
  faHourglass,
  faQuestion,
  faTriangleExclamation,
} from '@fortawesome/free-solid-svg-icons'
import { formatDateTimeNoSeconds, formatUptime } from '@/lib/dateTime'
import { useI18n } from 'vue-i18n'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import DataItem from '../DataItem.vue'
import { computed } from 'vue'

const { t, locale } = useI18n()
const { state: systemDetail } = useSystemDetail()
const { state: latestInventory } = useLatestInventory()

const uptimeLabel = computed(() => {
  return systemType.value === 'ns8'
    ? t('system_detail.leader_node_uptime')
    : t('system_detail.uptime')
})

const timezoneLabel = computed(() => {
  return systemType.value === 'ns8'
    ? t('system_detail.leader_node_timezone')
    : t('system_detail.timezone')
})

const systemType = computed(() => systemDetail.value.data?.type)

const leaderNode = computed(() => {
  const nodes = latestInventory.value.data?.data?.facts?.nodes

  if (!nodes) {
    return null
  }
  //// improve typing
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (Object.values(nodes).find((node: any) => node.cluster_leader === true) as any) ?? null
})

const uptimeSeconds = computed(() => {
  if (systemType.value === 'ns8') {
    return leaderNode.value?.uptime_seconds as number
  } else {
    return latestInventory.value.data?.data?.facts?.uptime_seconds as number | undefined
  }
})

const timezone = computed(() => {
  if (systemType.value === 'ns8') {
    return leaderNode.value?.timezone as string
  } else {
    return latestInventory.value.data?.data?.facts?.timezone as string | undefined
  }
})

const getBadgeKind = () => {
  switch (systemDetail.value.data?.heartbeat_status) {
    case 'active':
      return 'green'
    case 'inactive':
      return 'amber'
    case 'unknown':
      return 'gray'
    default:
      return 'indigo'
  }
}

const getBadgeIcon = () => {
  switch (systemDetail.value.data?.heartbeat_status) {
    case 'active':
      return faCheck
    case 'inactive':
      return faTriangleExclamation
    case 'unknown':
      return faHourglass
    default:
      return faQuestion
  }
}
</script>

<template>
  <NeCard>
    <div class="mb-4 flex items-center gap-4">
      <FontAwesomeIcon :icon="faCircleInfo" class="size-5 shrink-0" aria-hidden="true" />
      <NeHeading tag="h6">
        {{ $t('common.status').toUpperCase() }}
      </NeHeading>
    </div>
    <!-- get system detail error notification -->
    <NeInlineNotification
      v-if="systemDetail.status === 'error'"
      kind="error"
      :title="$t('system_detail.cannot_retrieve_system_detail')"
      :description="systemDetail.error.message"
      class="mb-6"
    />
    <!-- get latest inventory error notification -->
    <NeInlineNotification
      v-if="latestInventory.status === 'error'"
      kind="error"
      :title="$t('system_detail.cannot_retrieve_latest_inventory')"
      :description="latestInventory.error.message"
      class="mb-6"
    />
    <NeSkeleton
      v-else-if="latestInventory.status === 'pending' || systemDetail.status === 'pending'"
      :lines="6"
    />
    <div v-else class="divide-y divide-gray-200 dark:divide-gray-700">
      <!-- status -->
      <DataItem>
        <template #label>
          {{ $t('common.status') }}
        </template>
        <template #data>
          <NeBadgeV2 :kind="getBadgeKind()">
            <div class="flex items-center gap-1">
              <FontAwesomeIcon :icon="getBadgeIcon()" class="size-4" />
              {{ t(`systems.status_${systemDetail.data?.heartbeat_status}`) }}
            </div>
          </NeBadgeV2>
        </template>
      </DataItem>
      <!-- last inventory -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.last_inventory') }}
        </template>
        <template #data>
          {{
            latestInventory.data?.timestamp
              ? formatDateTimeNoSeconds(new Date(latestInventory.data?.timestamp), locale)
              : '-'
          }}
        </template>
      </DataItem>
      <!-- uptime -->
      <DataItem>
        <template #label>
          {{ uptimeLabel }}
        </template>
        <template #data>
          {{ uptimeSeconds ? formatUptime(uptimeSeconds, $t) : '-' }}
        </template>
      </DataItem>
      <!-- timezone -->
      <DataItem>
        <template #label>
          {{ timezoneLabel }}
        </template>
        <template #data>
          {{ timezone ? timezone : '-' }}
        </template>
      </DataItem>
    </div>
  </NeCard>
</template>
