<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later


  Add-ons view: the Report tab is the commercial overview of the add-ons the
  viewer can see (the whole fleet for the owner organization / a Super Admin,
  their own hierarchy for a distributor, reseller or customer). The
  Configuration tab manages the catalog of grantable types — a licensing
  back-office duty, so it stays owner-level: with a single tab left the tab bar
  itself is dropped.
-->

<script setup lang="ts">
import { NeHeading, NeTabs } from '@nethesis/vue-components'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabs } from '@/composables/useTabs'
import { isEntitlementAdmin } from '@/lib/permissions'
import EntitlementsCatalogPanel from '@/components/entitlements/EntitlementsCatalogPanel.vue'
import EntitlementsReportPanel from '@/components/entitlements/EntitlementsReportPanel.vue'

const { t } = useI18n()

const canConfigure = computed(() => isEntitlementAdmin())

const tabsConfig = computed(() => [
  { name: 'report', label: t('entitlements.report_tab') },
  ...(canConfigure.value
    ? [{ name: 'configuration', label: t('entitlements.configuration_tab') }]
    : []),
])

const { tabs, selectedTab } = useTabs(tabsConfig)
</script>

<template>
  <div>
    <!-- Header -->
    <NeHeading tag="h3" class="mb-7">{{ $t('entitlements-catalog.title') }}</NeHeading>

    <!-- Tab switcher (only when there is more than one tab to switch to) -->
    <NeTabs
      v-if="tabs.length > 1"
      :tabs="tabs"
      :selected="selectedTab"
      :sr-tabs-label="t('ne_tabs.tabs')"
      :sr-select-tab-label="t('ne_tabs.select_a_tab')"
      class="mb-8"
      @select-tab="selectedTab = $event"
    />

    <!-- Tab content: the catalog panel answers to the permission, never to the
       ?tab= query alone -->
    <EntitlementsCatalogPanel v-if="canConfigure && selectedTab === 'configuration'" />
    <EntitlementsReportPanel v-else />
  </div>
</template>
