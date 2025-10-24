<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { onMounted } from 'vue'
import { NeButton, NeHeading, NeSkeleton, NeTabs } from '@nethesis/vue-components'
import router from '@/router'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowLeft } from '@fortawesome/free-solid-svg-icons'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useTabs } from '@/composables/useTabs'
import { useI18n } from 'vue-i18n'
import SystemOverviewPanel from '@/components/systems/SystemOverviewPanel.vue'

const { t } = useI18n()
const { state } = useSystemDetail()
const { tabs, selectedTab } = useTabs([{ name: 'overview', label: t('system_detail.overview') }])

onMounted(() => {
  console.log('onMounted') ////
})

const goToSystems = () => {
  router.push({ name: 'systems' })
}
</script>

<template>
  <div>
    <NeButton kind="tertiary" size="sm" @click="goToSystems" class="mb-4 -ml-2">
      <template #prefix>
        <FontAwesomeIcon :icon="faArrowLeft" />
      </template>
      {{ $t('systems.title') }}
    </NeButton>
    <NeSkeleton v-if="state.status === 'pending'" size="lg" class="mb-9 w-xs" />
    <NeHeading v-else tag="h3" class="mb-7">
      {{ state.data?.name }}
    </NeHeading>
    <NeTabs
      :tabs="tabs"
      :selected="selectedTab"
      :sr-tabs-label="t('ne_tabs.tabs')"
      :sr-select-tab-label="t('ne_tabs.select_a_tab')"
      class="mb-8"
      @select-tab="selectedTab = $event"
    />
    <SystemOverviewPanel v-if="selectedTab === 'overview'" />
    <!-- {{ state.data }} //// -->
  </div>
</template>
