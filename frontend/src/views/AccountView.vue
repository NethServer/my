<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeHeading, NeTabs } from '@nethesis/vue-components'
import ImpersonationPanel from '@/components/account/impersonation/ImpersonationPanel.vue'
import { useTabs } from '@/composables/useTabs'
import { useI18n } from 'vue-i18n'
import GeneralPanel from '@/components/account/GeneralPanel.vue'

const { t } = useI18n()

const { tabs, selectedTab } = useTabs([
  { name: 'general', label: t('account.general') },
  { name: 'impersonation', label: t('account.impersonation.impersonation') },
])
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">
      {{ $t('account.title') }}
    </NeHeading>
    <NeTabs
      :tabs="tabs"
      :selected="selectedTab"
      :sr-tabs-label="t('ne_tabs.tabs')"
      :sr-select-tab-label="t('ne_tabs.select_a_tab')"
      class="mb-8"
      @select-tab="selectedTab = $event"
    />
    <GeneralPanel v-if="selectedTab === 'general'" />
    <ImpersonationPanel v-else-if="selectedTab === 'impersonation'" />
  </div>
</template>
