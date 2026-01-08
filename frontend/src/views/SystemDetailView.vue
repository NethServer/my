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
  NeTabs,
  NeTooltip,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowLeft, faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useTabs } from '@/composables/useTabs'
import { useI18n } from 'vue-i18n'
import SystemOverviewPanel from '@/components/systems/SystemOverviewPanel.vue'
import { useLatestInventory } from '@/queries/systems/latestInventory'
import { computed } from 'vue'

const { t } = useI18n()
const { state: systemDetail } = useSystemDetail()
const { state: latestInventory } = useLatestInventory()
const { tabs, selectedTab } = useTabs([{ name: 'overview', label: t('system_detail.overview') }])

const systemUrl = computed(() => {
  if (!systemDetail.value.data?.fqdn) {
    return ''
  }

  if (!['ns8', 'nsec'].includes(systemDetail.value.data?.type || '')) {
    return ''
  }

  const fqdn = systemDetail.value.data.fqdn
  let port = ''
  let path = ''

  if (systemDetail.value.data?.type === 'ns8') {
    path = '/cluster-admin'
  } else if (systemDetail.value.data?.type === 'nsec') {
    port = ':9090'
  }
  const url = `https://${fqdn}${port}${path}`
  return url
})

const openSystem = () => {
  if (systemUrl.value) {
    window.open(systemUrl.value, '_blank')
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
      <!-- open system -->
      <NeTooltip placement="left" trigger-event="mouseenter focus" class="shrink-0">
        <template #trigger>
          <NeButton kind="primary" :disabled="!systemUrl" @click="openSystem()">
            <template #prefix>
              <FontAwesomeIcon :icon="faArrowUpRightFromSquare" aria-hidden="true" />
            </template>
            {{ $t('system_detail.go_to_system') }}
          </NeButton>
        </template>
        <template #content>
          {{
            systemUrl
              ? $t('system_detail.go_to_system_tooltip')
              : $t('system_detail.cannot_determine_system_url_description')
          }}
        </template>
      </NeTooltip>
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
  </div>
</template>
