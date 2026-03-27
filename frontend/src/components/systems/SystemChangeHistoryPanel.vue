<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useInventoryTimeline } from '@/queries/systems/inventoryTimeline'
import { useI18n } from 'vue-i18n'
import UpdatingSpinner from '../UpdatingSpinner.vue'
import { NeInlineNotification } from '@nethesis/vue-components'
import SystemChangesTimeline from './SystemChangesTimeline.vue'

const { t } = useI18n()

const { state: timelineState, asyncStatus: timelineAsyncStatus } = useInventoryTimeline()
</script>

<template>
  <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
    <div class="max-w-2xl text-gray-500 dark:text-gray-400">
      {{ $t('system_detail.change_history_description') }}
    </div>
    <!-- update indicator -->
    <UpdatingSpinner
      v-if="timelineAsyncStatus === 'loading' && timelineState.status !== 'pending'"
    />
  </div>
  <!-- timeline error notification -->
  <NeInlineNotification
    v-if="timelineState.status === 'error'"
    kind="error"
    :title="t('system_detail.cannot_retrieve_inventory_timeline')"
    :description="timelineState.error.message"
    class="mb-6"
  />
  <!-- changes timeline -->
  <SystemChangesTimeline />
</template>
