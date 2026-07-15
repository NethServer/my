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
import { formatDateTimeNoSeconds, formatRelativeTime, formatUptime } from '@/lib/dateTime'
import { useI18n } from 'vue-i18n'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useSystemActiveAlerts } from '@/queries/systems/activeAlerts'
import { useSystemBackups } from '@/queries/systems/backups'
import DataItem from '../common/DataItem.vue'
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import SystemStatusIcon from './SystemStatusIcon.vue'
import type { Ns8Facts } from '@/lib/systems/ns8Facts'
import type { NsecFacts } from '@/lib/systems/nsecFacts'

const { t, locale } = useI18n()
const route = useRoute()
const { state: systemDetail } = useSystemDetail()
const { state: latestInventory } = useLatestInventory()
const { state: activeAlerts } = useSystemActiveAlerts()
const { state: systemBackups } = useSystemBackups()

const activeAlertsCount = computed(() => activeAlerts.value?.data?.alerts?.length ?? 0)
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
    <div class="mb-4 flex h-10 items-center gap-4">
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
              <NeTooltip
                v-if="
                  systemDetail.data?.status === 'active' ||
                  systemDetail.data?.status === 'inactive' ||
                  systemDetail.data?.status === 'unknown'
                "
                trigger-event="mouseenter focus"
                placement="left"
              >
                <template #trigger>
                  <div class="flex items-center gap-2">
                    <SystemStatusIcon :status="systemDetail.data?.status" />
                    {{ t(`systems.status_${systemDetail.data?.status}`) }}
                  </div>
                </template>
                <template #content>
                  <template v-if="systemDetail.data?.status === 'unknown'">
                    {{ $t('system_detail.no_heartbeat_yet') }}
                  </template>
                  <template v-else>
                    {{
                      $t('system_detail.last_heartbeat_time', {
                        time: formatRelativeTime(systemDetail.data?.last_heartbeat ?? '', locale),
                      })
                    }}
                  </template>
                </template>
              </NeTooltip>
              <template v-else>
                <SystemStatusIcon :status="systemDetail.data?.status" />
                {{ t(`systems.status_${systemDetail.data?.status}`) }}
              </template>
            </template>
            <span v-else>-</span>
          </div>
        </template>
      </DataItem>
      <!-- last inventory -->
      <DataItem>
        <template #label>
          {{ $t('system_detail.last_inventory') }}
        </template>
        <template #data>
          <div v-if="!latestInventory.data?.timestamp" class="flex items-center gap-2">
            <!-- no inventory warning -->
            <FontAwesomeIcon
              :icon="faTriangleExclamation"
              class="size-4 text-amber-700 dark:text-amber-500"
              aria-hidden="true"
            />
            {{ $t('system_detail.no_inventory_yet') }}
          </div>
          <NeTooltip v-else trigger-event="mouseenter focus" placement="left">
            <template #trigger>
              {{ formatRelativeTime(latestInventory.data?.timestamp, locale) }}
            </template>
            <template #content>
              {{ formatDateTimeNoSeconds(new Date(latestInventory.data?.timestamp), locale) }}
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
          {{ $t('alerts.title') }}
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
              <router-link
                :to="{
                  name: 'alerts',
                  query: {
                    system_key: systemDetail.data?.system_key,
                    system_name: systemDetail.data?.name,
                  },
                }"
                class="flex items-center gap-2 hover:underline"
                :aria-label="
                  $t('system_detail.show_system_alerts', { name: systemDetail.data?.name })
                "
              >
                <FontAwesomeIcon
                  :icon="faTriangleExclamation"
                  class="size-4 text-amber-700 dark:text-amber-500"
                  aria-hidden="true"
                />
                {{
                  $t('system_detail.n_active_alerts', { n: activeAlertsCount }, activeAlertsCount)
                }}
              </router-link>
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
            <template v-else>
              <router-link
                :to="{
                  name: 'system_detail',
                  params: { systemId: route.params.systemId },
                  query: { tab: 'backups' },
                }"
                class="flex items-center gap-2 hover:underline"
                :aria-label="
                  $t('system_detail.show_system_backups', { name: systemDetail.data?.name })
                "
              >
                <template v-if="hasBackups">
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
              </router-link>
            </template>
          </div>
        </template>
      </DataItem>
    </div>
  </NeCard>
</template>
