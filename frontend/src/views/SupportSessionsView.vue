<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed, ref } from 'vue'
import { NeButton, NeExpandable, NeHeading } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faXmark } from '@fortawesome/free-solid-svg-icons'
import SupportSessionsTable from '@/components/support/SupportSessionsTable.vue'
import CloseAllSessionsModal from '@/components/support/CloseAllSessionsModal.vue'
import {
  useLiveSupportSessions,
  useSupportSessionsHistory,
} from '@/queries/support/supportSessions'

const { isEnabled: isHistoryEnabled } = useSupportSessionsHistory()
const { state: liveState, areDefaultFiltersApplied } = useLiveSupportSessions()

const isCloseAllShown = ref(false)

// a filtered empty list may still hide live sessions
const hasNoLiveSessions = computed(
  () => areDefaultFiltersApplied.value && liveState.value.data?.pagination?.total_count === 0,
)

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
        <div class="mb-6 flex flex-wrap items-center justify-between gap-4">
          <h4 class="text-base font-medium text-gray-900 dark:text-gray-100">
            {{ $t('support.live_sessions_title') }}
          </h4>
          <NeButton
            kind="secondary"
            size="sm"
            :disabled="hasNoLiveSessions"
            @click="isCloseAllShown = true"
          >
            <template #prefix>
              <FontAwesomeIcon :icon="faXmark" aria-hidden="true" />
            </template>
            {{ $t('support.close_all_sessions') }}
          </NeButton>
        </div>
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
    <CloseAllSessionsModal :visible="isCloseAllShown" @close="isCloseAllShown = false" />
  </div>
</template>
