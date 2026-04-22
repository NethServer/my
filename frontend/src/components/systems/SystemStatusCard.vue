<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeCard,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
  NeTooltip,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { useLatestInventory } from '@/queries/systems/latestInventory'
import {
  faCircleCheck,
  faCircleInfo,
  faTriangleExclamation,
} from '@fortawesome/free-solid-svg-icons'
import { formatDateTimeNoSeconds, formatTimeAgo, formatUptime } from '@/lib/dateTime'
import { useI18n } from 'vue-i18n'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useSystemActiveAlerts } from '@/queries/systems/activeAlerts'
import { useSystemBackups } from '@/queries/systems/backups'
import DataItem from '../DataItem.vue'
import { computed } from 'vue'
import SystemStatusIcon from './SystemStatusIcon.vue'
import type { Ns8Facts } from '@/lib/systems/ns8Facts'
import type { NsecFacts } from '@/lib/systems/nsecFacts'

const { t, locale } = useI18n()
const { state: systemDetail } = useSystemDetail()
const { state: latestInventory } = useLatestInventory()
const { state: activeAlerts } = useSystemActiveAlerts()
const { state: systemBackups } = useSystemBackups()

const activeAlertsCount = computed(() => activeAlerts.value.data?.length ?? 0)
const backupsCount = computed(() => systemBackups.value.data?.backups?.length ?? 0)
const hasActiveAlerts = computed(() => activeAlertsCount.value > 0)
const hasBackups = computed(() => backupsCount.value > 0)

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
  if (systemType.value !== 'ns8') {
    return null
  }

  const facts = latestInventory.value.data?.data?.facts as Ns8Facts | undefined
  const nodes = facts?.nodes

  if (!nodes) {
    return null
  }
  return Object.values(nodes).find((node) => node.cluster_leader === true)
})

const uptimeSeconds = computed(() => {
  if (systemType.value === 'ns8') {
    return leaderNode.value?.uptime_seconds
  } else {
    const facts = latestInventory.value.data?.data?.facts as NsecFacts | undefined
    return facts?.uptime_seconds
  }
})

const timezone = computed(() => {
  if (systemType.value === 'ns8') {
    return leaderNode.value?.timezone as string
  } else {
    const facts = latestInventory.value.data?.data?.facts as NsecFacts | undefined
    return facts?.timezone
  }
})
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
          <div class="flex items-center gap-2">
            <template v-if="systemDetail.data?.status">
              <SystemStatusIcon :status="systemDetail.data?.status" />
              {{ t(`systems.status_${systemDetail.data?.status}`) }}
            </template>
            <span v-else>-</span>
            <!-- no inventory warning (do not show for pending/unknown status) -->
            <NeTooltip
              v-if="
                latestInventory.status === 'success' &&
                !latestInventory.data &&
                systemDetail.data?.status !== 'unknown'
              "
              trigger-event="mouseenter focus"
              placement="top"
            >
              <template #trigger>
                <FontAwesomeIcon
                  :icon="faTriangleExclamation"
                  class="size-4 text-amber-700 dark:text-amber-500"
                  aria-hidden="true"
                />
              </template>
              <template #content>
                {{ $t('system_detail.no_inventory_available') }}
              </template>
            </NeTooltip>
          </div>
        </template>
      </DataItem>
      <!-- last inventory -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.last_inventory') }}
        </template>
        <template #data>
          <NeTooltip trigger-event="mouseenter focus" placement="left">
            <template #trigger>
              {{ formatTimeAgo(latestInventory.data?.timestamp, $t) }}
            </template>
            <template #content>
              {{
                latestInventory.data?.timestamp
                  ? formatDateTimeNoSeconds(new Date(latestInventory.data?.timestamp), locale)
                  : '-'
              }}
            </template>
          </NeTooltip>
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
      <!-- alerts indicator -->
      <DataItem>
        <template #label>
          {{ $t('alerting.title') }}
        </template>
        <template #data>
          <div class="flex items-center gap-2">
            <template v-if="activeAlerts.status === 'pending'">
              <span>-</span>
            </template>
            <template v-else-if="activeAlerts.status === 'error'">
              <span>-</span>
            </template>
            <template v-else-if="hasActiveAlerts">
              <FontAwesomeIcon
                :icon="faTriangleExclamation"
                class="size-4 text-amber-700 dark:text-amber-500"
                aria-hidden="true"
              />
              {{ $t('system_detail.n_active_alerts', { n: activeAlertsCount }, activeAlertsCount) }}
            </template>
            <template v-else>
              <FontAwesomeIcon
                :icon="faCircleCheck"
                class="size-4 text-green-700 dark:text-green-500"
                aria-hidden="true"
              />
              {{ $t('system_detail.no_active_alerts') }}
            </template>
          </div>
        </template>
      </DataItem>
      <!-- backups indicator -->
      <DataItem>
        <template #label>
          {{ $t('backups.title') }}
        </template>
        <template #data>
          <div class="flex items-center gap-2">
            <template v-if="systemBackups.status === 'pending'">
              <span>-</span>
            </template>
            <template v-else-if="systemBackups.status === 'error'">
              <span>-</span>
            </template>
            <template v-else-if="hasBackups">
              <FontAwesomeIcon
                :icon="faCircleCheck"
                class="size-4 text-green-700 dark:text-green-500"
                aria-hidden="true"
              />
              {{ $t('system_detail.n_backups_stored', { n: backupsCount }, backupsCount) }}
            </template>
            <template v-else>
              <FontAwesomeIcon
                :icon="faTriangleExclamation"
                class="size-4 text-amber-700 dark:text-amber-500"
                aria-hidden="true"
              />
              {{ $t('system_detail.no_backups_stored') }}
            </template>
          </div>
        </template>
      </DataItem>
    </div>
  </NeCard>
</template>
