<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeBadgeV2,
  NeTable,
  NeTableHead,
  NeTableHeadCell,
  NeTableBody,
  NeTableRow,
  NeTableCell,
  NePaginator,
  NeEmptyState,
  NeInlineNotification,
  NeSkeleton,
  type NeBadgeV2Kind,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faBell } from '@fortawesome/free-solid-svg-icons'
import { useAlertHistory } from '@/queries/systems/alertHistory'
import { useI18n } from 'vue-i18n'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import UpdatingSpinner from '@/components/UpdatingSpinner.vue'

const { locale } = useI18n()
const { state, asyncStatus, pageNum, pageSize } = useAlertHistory()

function getSeverityBadgeKind(severity: string | null | undefined): NeBadgeV2Kind {
  switch (severity?.toLowerCase()) {
    case 'critical':
      return 'rose'
    case 'warning':
      return 'amber'
    case 'info':
      return 'blue'
    default:
      return 'gray'
  }
}
</script>

<template>
  <div>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('alerting.history_tab_description') }}
      </div>
      <UpdatingSpinner v-if="asyncStatus === 'loading' && state.status !== 'pending'" />
    </div>

    <!-- error -->
    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="$t('alerting.cannot_retrieve_alert_history')"
      :description="state.error?.message"
      class="mb-6"
    />

    <!-- loading skeleton -->
    <NeSkeleton v-else-if="state.status === 'pending'" :lines="8" />

    <!-- empty state -->
    <NeEmptyState
      v-else-if="!state.data?.alerts.length"
      :title="$t('alerting.no_alert_history')"
      :description="$t('alerting.no_alert_history_description')"
      :icon="faBell"
    />

    <!-- table -->
    <template v-else>
      <NeTable :aria-label="$t('alerting.alert_history')" card-breakpoint="md">
        <NeTableHead>
          <NeTableHeadCell>{{ $t('alerting.alertname') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('alerting.severity') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('common.status') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('alerting.summary') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('alerting.starts_at') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('alerting.ends_at') }}</NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="alert in state.data.alerts" :key="alert.id">
            <NeTableCell :data-label="$t('alerting.alertname')">
              <div class="flex items-center gap-2">
                <FontAwesomeIcon
                  :icon="faBell"
                  class="h-4 w-4 shrink-0 text-gray-400"
                  aria-hidden="true"
                />
                <span class="font-medium">{{ alert.alertname }}</span>
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('alerting.severity')">
              <NeBadgeV2
                v-if="alert.severity"
                :kind="getSeverityBadgeKind(alert.severity)"
                size="xs"
              >
                {{ alert.severity }}
              </NeBadgeV2>
              <span v-else>-</span>
            </NeTableCell>
            <NeTableCell :data-label="$t('common.status')">
              <NeBadgeV2 kind="gray" size="xs">{{ alert.status }}</NeBadgeV2>
            </NeTableCell>
            <NeTableCell :data-label="$t('alerting.summary')">
              {{ alert.summary || alert.annotations?.summary || '-' }}
            </NeTableCell>
            <NeTableCell :data-label="$t('alerting.starts_at')">
              {{
                alert.starts_at ? formatDateTimeNoSeconds(new Date(alert.starts_at), locale) : '-'
              }}
            </NeTableCell>
            <NeTableCell :data-label="$t('alerting.ends_at')">
              {{ alert.ends_at ? formatDateTimeNoSeconds(new Date(alert.ends_at), locale) : '-' }}
            </NeTableCell>
          </NeTableRow>
        </NeTableBody>
      </NeTable>

      <!-- pagination -->
      <NePaginator
        v-if="state.data.pagination"
        :current-page="pageNum"
        :total-rows="state.data.pagination.total_count"
        :page-size="pageSize"
        :page-sizes="[5, 10, 25, 50, 100]"
        :nav-pagination-label="$t('ne_table.pagination')"
        :next-label="$t('ne_table.go_to_next_page')"
        :previous-label="$t('ne_table.go_to_previous_page')"
        :range-of-total-label="$t('ne_table.of')"
        :page-size-label="$t('ne_table.show')"
        class="mt-6"
        @select-page="pageNum = $event"
        @select-page-size="pageSize = $event"
      />
    </template>
  </div>
</template>
