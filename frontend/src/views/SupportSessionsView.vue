<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { ref } from 'vue'
import { NeExpandable, NeHeading } from '@nethesis/vue-components'
import SupportSessionsTable from '@/components/support/SupportSessionsTable.vue'
import { useSupportSessionsHistory } from '@/queries/support/supportSessions'

const { isEnabled: isHistoryEnabled } = useSupportSessionsHistory()

// the history loads only once its section is opened
const isHistoryExpanded = ref(false)

function onSetExpanded(expanded: boolean) {
  isHistoryExpanded.value = expanded
  isHistoryEnabled.value = expanded
}
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('support.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('support.page_description') }}
      </div>
    </div>
    <div class="space-y-10">
      <section>
        <h4 class="mb-6 text-base font-medium text-gray-900 dark:text-gray-100">
          {{ $t('support.live_sessions_title') }}
        </h4>
        <SupportSessionsTable scope="live" />
      </section>
      <NeExpandable
        :label="$t('support.sessions_history_title')"
        :is-expanded="isHistoryExpanded"
        @set-expanded="onSetExpanded"
      >
        <SupportSessionsTable v-if="isHistoryExpanded" scope="history" class="mt-6" />
      </NeExpandable>
    </div>
  </div>
</template>
