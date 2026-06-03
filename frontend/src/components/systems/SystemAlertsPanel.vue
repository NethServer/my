<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faChevronDown, faChevronUp } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { ref } from 'vue'
import SystemActiveAlertsTable from '@/components/systems/SystemActiveAlertsTable.vue'
import SystemHistoryAlertsTable from '@/components/systems/SystemHistoryAlertsTable.vue'

const isHistoryExpanded = ref(false)
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
    <section>
      <!-- Collapsible header -->
      <button
        class="hover:bg-elevation-2 flex w-full items-center gap-3 rounded-lg px-4 py-2 text-left"
        :aria-expanded="isHistoryExpanded"
        @click="isHistoryExpanded = !isHistoryExpanded"
      >
        <FontAwesomeIcon
          :icon="isHistoryExpanded ? faChevronUp : faChevronDown"
          class="h-4 w-4 shrink-0 text-gray-500 dark:text-gray-400"
          aria-hidden="true"
        />
        <div>
          <h4 class="text-base font-medium text-gray-900 dark:text-gray-100">
            {{ $t('system_detail.alert_history_title') }}
          </h4>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ $t('system_detail.alert_history_description') }}
          </p>
        </div>
      </button>

      <!-- Expanded content -->
      <div v-if="isHistoryExpanded" class="mt-6">
        <SystemHistoryAlertsTable />
      </div>
    </section>
  </div>
</template>
