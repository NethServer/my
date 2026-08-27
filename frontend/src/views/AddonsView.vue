<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeHeading, NeTabs } from '@nethesis/vue-components'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTabs } from '@/composables/useTabs'
import AddonsCatalogPanel from '@/components/addons/AddonsCatalogPanel.vue'
import AddonsReportPanel from '@/components/addons/AddonsReportPanel.vue'

const { t } = useI18n()

const tabsConfig = computed(() => [
  { name: 'report', label: t('addons.report_tab') },
  { name: 'catalog', label: t('addons.catalog_tab') },
])

const { tabs, selectedTab } = useTabs(tabsConfig, 'report')
</script>

<template>
  <div>
    <!-- Header -->
    <NeHeading tag="h3" class="mb-7">{{ $t('addons.title') }}</NeHeading>

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
    <AddonsReportPanel v-if="selectedTab === 'report'" />
    <AddonsCatalogPanel v-else />
  </div>
</template>
