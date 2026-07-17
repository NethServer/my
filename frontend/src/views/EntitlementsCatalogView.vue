<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later


  Add-ons view for the owner organization / Super Admins: the Report tab is
  the commercial overview of the fleet (where to act), the Configuration tab
  manages the catalog of grantable types (a once-in-a-while operation).
-->

<script setup lang="ts">
import { NeHeading, NeTabs } from '@nethesis/vue-components'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabs } from '@/composables/useTabs'
import EntitlementsCatalogPanel from '@/components/entitlements/EntitlementsCatalogPanel.vue'
import EntitlementsReportPanel from '@/components/entitlements/EntitlementsReportPanel.vue'

const { t } = useI18n()

const tabsConfig = computed(() => [
  { name: 'report', label: t('entitlements.report_tab') },
  { name: 'configuration', label: t('entitlements.configuration_tab') },
])

const { tabs, selectedTab } = useTabs(tabsConfig)
</script>

<template>
  <div>
    <!-- Header -->
    <NeHeading tag="h3" class="mb-7">{{ $t('entitlements-catalog.title') }}</NeHeading>

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
    <EntitlementsReportPanel v-if="selectedTab === 'report'" />
    <EntitlementsCatalogPanel v-else-if="selectedTab === 'configuration'" />
  </div>
</template>
