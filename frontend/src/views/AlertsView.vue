<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeHeading, NeTabs } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useTabs } from '@/composables/useTabs'
import { canReadAlerts } from '@/lib/permissions'
import ActiveAlertsPanel from '@/components/alerts/ActiveAlertsPanel.vue'
import AlertNotificationsPanel from '@/components/alerts/AlertNotificationsPanel.vue'
import { computed } from 'vue'

const { t } = useI18n()

const tabsConfig = computed(() => [
  { name: 'active_alerts', label: t('alerts.active_alerts_tab') },
  ...(canReadAlerts() ? [{ name: 'notifications', label: t('alerts.notifications_tab') }] : []),
])

const { tabs, selectedTab } = useTabs(tabsConfig)
</script>

<template>
  <div>
    <!-- Header -->
    <NeHeading tag="h3" class="mb-7">{{ $t('alerts.title') }}</NeHeading>

    <!-- Tab switcher -->
    <NeTabs
      :tabs="tabs"
      :selected="selectedTab"
      :sr-tabs-label="t('ne_tabs.tabs')"
      :sr-select-tab-label="t('ne_tabs.select_a_tab')"
      class="mb-8"
      @select-tab="selectedTab = $event"
    />

    <!-- Tab content -->
    <ActiveAlertsPanel v-if="selectedTab === 'active_alerts'" />
    <AlertNotificationsPanel v-else-if="selectedTab === 'notifications'" />
  </div>
</template>
