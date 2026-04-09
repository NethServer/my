<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
  NeSpinner,
  NeTabs,
  NeTooltip,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowLeft, faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useTabs } from '@/composables/useTabs'
import { useI18n } from 'vue-i18n'
import SystemOverviewPanel from '@/components/systems/SystemOverviewPanel.vue'
import SystemChangeHistoryPanel from '@/components/systems/SystemChangeHistoryPanel.vue'
import SystemAlertHistoryPanel from '@/components/systems/SystemAlertHistoryPanel.vue'
import { useLatestInventory } from '@/queries/systems/latestInventory'
import { useSystemReachability } from '@/queries/systems/systemReachability'
import { computed } from 'vue'

const { t } = useI18n()
const { state: systemDetail } = useSystemDetail()
const { state: latestInventory } = useLatestInventory()
const { state: reachabilityState, asyncStatus: reachabilityAsyncStatus } = useSystemReachability()
const { tabs, selectedTab } = useTabs([
  { name: 'overview', label: t('system_detail.overview') },
  { name: 'change_history', label: t('system_detail.change_history') },
  { name: 'alert_history', label: t('alerting.alert_history') },
])

const isSystemReachable = computed(() => !!reachabilityState.value.data?.reachable)
const isCheckingReachability = computed(() => reachabilityAsyncStatus.value === 'loading')
const isGoToSystemDisabled = computed(
  () => isCheckingReachability.value || !isSystemReachable.value,
)

const openSystem = () => {
  const url = reachabilityState.value.data?.url
  if (url) {
    window.open(url, '_blank')
  }
}
</script>

<template>
  <div>
    <router-link to="/systems">
      <NeButton kind="tertiary" size="sm" class="mb-4 -ml-2">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowLeft" />
        </template>
        {{ $t('systems.title') }}
      </NeButton>
    </router-link>
    <!-- get system detail error notification -->
    <NeInlineNotification
      v-if="systemDetail.status === 'error'"
      kind="error"
      :title="$t('system_detail.cannot_retrieve_system_detail')"
      :description="systemDetail.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="systemDetail.status === 'pending'" size="lg" class="mb-9 w-xs" />
    <div v-else class="flex items-start justify-between gap-4">
      <NeHeading tag="h3" class="mb-7">
        {{ systemDetail.data?.name }}
      </NeHeading>
      <div class="flex shrink-0 items-center gap-2">
        <NeSpinner v-if="reachabilityState.status === 'pending'" color="white" />
        <!-- open system (with tooltip only when not reachable) -->
        <NeTooltip
          v-if="!isSystemReachable"
          placement="left"
          trigger-event="mouseenter focus"
          class="shrink-0"
        >
          <template #trigger>
            <NeButton kind="primary" :disabled="isGoToSystemDisabled">
              <template #prefix>
                <FontAwesomeIcon :icon="faArrowUpRightFromSquare" aria-hidden="true" />
              </template>
              {{ $t('system_detail.go_to_system') }}
            </NeButton>
          </template>
          <template #content>
            {{
              isCheckingReachability
                ? $t('system_detail.checking_reachability')
                : $t('system_detail.system_unreachable')
            }}
          </template>
        </NeTooltip>
        <NeButton
          v-else
          kind="primary"
          :disabled="isGoToSystemDisabled"
          class="shrink-0"
          @click="openSystem()"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="faArrowUpRightFromSquare" aria-hidden="true" />
          </template>
          {{ $t('system_detail.go_to_system') }}
        </NeButton>
      </div>
    </div>
    <!-- no inventory notification -->
    <NeInlineNotification
      v-if="latestInventory.status === 'success' && !latestInventory.data"
      kind="warning"
      :title="$t('system_detail.no_inventory_available')"
      :description="$t('system_detail.no_inventory_available_description')"
      class="mb-4"
    />
    <NeTabs
      :tabs="tabs"
      :selected="selectedTab"
      :sr-tabs-label="t('ne_tabs.tabs')"
      :sr-select-tab-label="t('ne_tabs.select_a_tab')"
      class="mb-8"
      @select-tab="selectedTab = $event"
    />
    <SystemOverviewPanel v-if="selectedTab === 'overview'" />
    <SystemChangeHistoryPanel v-else-if="selectedTab === 'change_history'" />
    <SystemAlertHistoryPanel v-else-if="selectedTab === 'alert_history'" />
  </div>
</template>
