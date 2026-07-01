<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { ref } from 'vue'
import SystemActiveAlertsTable from '@/components/systems/SystemActiveAlertsTable.vue'
import SystemHistoryAlertsTable from '@/components/systems/SystemHistoryAlertsTable.vue'
import { useI18n } from 'vue-i18n'
import { NeExpandable } from '@nethesis/vue-components'
import { useSystemAlertHistory } from '@/queries/systemAlerts/systemAlertHistory'

const { t } = useI18n()
const { isHistoryEnabled } = useSystemAlertHistory()

const isHistoryExpanded = ref(false)

function onSetExpanded(ev: boolean) {
  isHistoryExpanded.value = ev
  isHistoryEnabled.value = ev
}
</script>

<template>
  <div class="space-y-10">
    <!-- ── Active alerts section ──────────────────────────────────────────── -->
    <section>
      <div class="mb-8">
        <h4 class="text-base font-medium text-gray-900 dark:text-gray-100">
          {{ $t('system_detail.active_alerts_title') }}
        </h4>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ $t('system_detail.active_alerts_description') }}
        </p>
      </div>
      <SystemActiveAlertsTable />
    </section>

    <!-- ── Alert history section (collapsible) ────────────────────────────── -->
    <NeExpandable
      :label="t('system_detail.alert_history_title')"
      :is-expanded="isHistoryExpanded"
      @set-expanded="onSetExpanded"
    >
      <SystemHistoryAlertsTable class="mt-6" />
    </NeExpandable>
  </div>
</template>
