<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faUserSecret } from '@fortawesome/free-solid-svg-icons'
import {
  NeTable,
  NeTableHead,
  NeTableHeadCell,
  NeTableBody,
  NeTableRow,
  NeTableCell,
  NePaginator,
  NeButton,
  NeEmptyState,
  NeInlineNotification,
} from '@nethesis/vue-components'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import { useImpersonationSessions } from '@/queries/impersonationSessions'
import { SESSIONS_TABLE_ID, type Session } from '@/lib/impersonationSessions'
import UpdatingSpinner from '@/components/UpdatingSpinner.vue'
import { formatDateTimeNoSeconds, formatMinutes } from '@/lib/dateTime'

const { t, locale } = useI18n()
const { state, asyncStatus, pageNum, pageSize /*sortBy, sortDescending */ } =
  useImpersonationSessions()

const sessionsPage = computed(() => {
  return state.value.data?.sessions || []
})

const pagination = computed(() => {
  return state.value.data?.pagination
})

////
// const onSort = (payload: SortEvent) => {
//   sortBy.value = payload.key as keyof Session
//   sortDescending.value = payload.descending
// }

const showSessionModal = (session: Session) => {
  console.log('showSessionModal', session) ////
}
</script>

<template>
  <div>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('account.impersonation.sessions_description') }}
      </div>
      <!-- update indicator -->
      <UpdatingSpinner v-if="asyncStatus === 'loading' && state.status !== 'pending'" />
    </div>
    <div class="flex flex-col gap-6">
      <!-- get sessions error notification -->
      <NeInlineNotification
        v-if="state.status === 'error'"
        kind="error"
        :title="$t('account.impersonation.cannot_retrieve_impersonation_sessions')"
        :description="state.error.message"
      />
      <!-- <NeSortDropdown ////
        v-model:sort-key="sortBy"
        v-model:sort-descending="sortDescending"
        :label="t('sort.sort')"
        :options="[
          { id: 'start_time', label: t('account.session_start') },
          { id: 'end_time', label: t('account.session_end') },
          { id: 'duration_minutes', label: t('account.duration') },
          { id: 'impersonator_username', label: t('account.impersonator') },
          { id: 'status', label: t('account.session_status') },
        ]"
        :open-menu-aria-label="t('ne_dropdown.open_menu')"
        :sort-by-label="t('sort.sort_by')"
        :sort-direction-label="t('sort.direction')"
        :ascending-label="t('sort.ascending')"
        :descending-label="t('sort.descending')"
        class="xl:hidden"
      /> -->
      <NeEmptyState
        v-if="!state.data?.sessions.length && state.status !== 'pending'"
        :title="$t('account.impersonation.no_sessions')"
        :description="$t('account.impersonation.no_sessions_description')"
        :icon="faUserSecret"
        class="bg-white dark:bg-gray-950"
      />
      <!-- //// check breakpoint, skeleton-columns -->
      <NeTable
        :aria-label="$t('account.impersonation.sessions')"
        card-breakpoint="xl"
        :loading="state.status === 'pending'"
        :skeleton-columns="7"
        :skeleton-rows="7"
      >
        <NeTableHead>
          <NeTableHeadCell>{{ $t('account.impersonation.session_start') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('account.impersonation.session_end') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('account.impersonation.duration') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('account.impersonation.impersonator') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('account.impersonation.session_status') }}</NeTableHeadCell>
          <NeTableHeadCell>
            <!-- no header for actions -->
          </NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="(item, index) in sessionsPage" :key="index">
            <NeTableCell :data-label="$t('account.impersonation.session_start')">
              {{
                item.start_time ? formatDateTimeNoSeconds(new Date(item.start_time), locale) : '-'
              }}
            </NeTableCell>
            <NeTableCell :data-label="$t('account.impersonation.session_end')">
              {{ item.end_time ? formatDateTimeNoSeconds(new Date(item.end_time), locale) : '-' }}
            </NeTableCell>
            <NeTableCell :data-label="$t('account.impersonation.duration')">
              <span v-if="item.duration_minutes">{{
                item.duration_minutes ? formatMinutes(item.duration_minutes, $t) : '-'
              }}</span>
              <span v-else-if="item.duration_minutes == 0">{{
                $t('account.impersonation.less_than_a_minute')
              }}</span>
              <span v-else>-</span>
            </NeTableCell>
            <NeTableCell :data-label="$t('account.impersonation.impersonator')">
              {{ item.impersonator_username || '-' }}
            </NeTableCell>
            <NeTableCell :data-label="$t('account.impersonation.session_status')">
              {{ t(`account.impersonation.status_${item.status}`) || '-' }}
            </NeTableCell>
            <NeTableCell :data-label="$t('common.actions')">
              <div class="-ml-2.5 flex gap-2 xl:ml-0 xl:justify-end">
                <NeButton kind="tertiary" @click="showSessionModal(item)">
                  {{ $t('common.show') }}
                </NeButton>
              </div>
            </NeTableCell>
          </NeTableRow>
        </NeTableBody>
        <template #paginator>
          <NePaginator
            :current-page="pageNum"
            :total-rows="pagination?.total_count || 0"
            :page-size="pageSize"
            :page-sizes="[5, 10, 25, 50, 100]"
            :nav-pagination-label="$t('ne_table.pagination')"
            :next-label="$t('ne_table.go_to_next_page')"
            :previous-label="$t('ne_table.go_to_previous_page')"
            :range-of-total-label="$t('ne_table.of')"
            :page-size-label="$t('ne_table.show')"
            @select-page="
              (page: number) => {
                pageNum = page
              }
            "
            @select-page-size="
              (size: number) => {
                pageSize = size
                savePageSizeToStorage(SESSIONS_TABLE_ID, size)
              }
            "
          />
        </template>
      </NeTable>
    </div>
  </div>
</template>
