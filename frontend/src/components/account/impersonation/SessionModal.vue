<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeLink,
  NeModal,
  NePaginator,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  NeTooltip,
} from '@nethesis/vue-components'
import {
  useImpersonationSessionAudit,
  useImpersonationSessionAuditStore,
} from '@/queries/impersonationSessionAudit'
import { formatDateTime, formatDateTimeNoSeconds, formatMinutes } from '@/lib/dateTime'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { SESSION_AUDIT_TABLE_ID } from '@/lib/impersonationSessions'
import { savePageSizeToStorage } from '@/lib/tablePageSize'

const { visible = false } = defineProps<{
  visible: boolean
}>()

const emit = defineEmits(['close'])

const { t, locale } = useI18n()
const sessionAuditStore = useImpersonationSessionAuditStore()
const {
  state,
  asyncStatus,
  pageNum,
  pageSize,
  // sortBy, ////
  // sortDescending, ////
} = useImpersonationSessionAudit()
const requestDataJustCopied = ref(false)

const sessionsPage = computed(() => {
  return state.value.data?.entries || []
})

const pagination = computed(() => {
  return state.value.data?.pagination
})

const session = computed(() => sessionAuditStore.session)

const prettyPrintJsonString = (jsonString: string, truncate = false) => {
  const maxLines = 15
  try {
    // Parse once to get the actual JSON object
    const parsed = JSON.parse(jsonString)

    if (!truncate) {
      // If no truncation needed, return pretty-printed JSON
      return JSON.stringify(parsed, null, 2)
    }

    // Stringify with formatting, truncating if necessary
    const formatted = JSON.stringify(parsed, null, 2)

    // Split into lines
    const lines = formatted.split('\n')

    // If more than 8 lines, truncate and add ellipsis
    if (lines.length > maxLines) {
      return lines.slice(0, maxLines).join('\n') + '\n...'
    }
    return formatted
  } catch (e) {
    console.error('Invalid JSON')
  }
}

const copyRequestDataToClipboard = (jsonString: string) => {
  const formattedRequestData = prettyPrintJsonString(jsonString, false) || ''
  navigator.clipboard.writeText(formattedRequestData).then(
    () => {
      console.log('Request data copied to clipboard') ////
    },
    (err) => {
      console.error('Could not copy text: ', err)
      requestDataJustCopied.value = false
    },
  )

  // show "Copied" for 3 seconds
  requestDataJustCopied.value = true
  setTimeout(() => {
    requestDataJustCopied.value = false
  }, 3000)
}
</script>

<template>
  <NeModal
    :visible="visible"
    size="xxl"
    :title="$t('account.impersonation.impersonation_audit_log')"
    :primary-label="$t('common.close')"
    cancel-label=""
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="emit('close')"
  >
    <!-- sessionAuditStore {{ sessionAuditStore.session }} //// -->

    <!-- <div>state.data {{ state.data }} ////</div> -->

    <div class="mt-6 mb-6 grid grid-cols-1 gap-x-6 gap-y-2 sm:grid-cols-2 lg:grid-cols-3">
      <div>
        <span class="font-medium">{{ $t('account.impersonation.session_start') }}:</span>
        {{
          session?.start_time ? formatDateTimeNoSeconds(new Date(session.start_time), locale) : '-'
        }}
      </div>
      <div>
        <span class="font-medium">{{ $t('account.impersonation.session_end') }}:</span>
        {{ session?.end_time ? formatDateTimeNoSeconds(new Date(session?.end_time), locale) : '-' }}
      </div>
      <div>
        <span class="font-medium">{{ $t('account.impersonation.duration') }}: </span>
        <span class="font-medium"></span
        ><span v-if="session?.duration_minutes">{{
          session?.duration_minutes ? formatMinutes(session?.duration_minutes, $t) : '-'
        }}</span>
        <span v-else-if="session?.duration_minutes == 0">{{
          $t('account.impersonation.less_than_a_minute')
        }}</span>
        <span v-else>-</span>
      </div>
      <div>
        <span class="font-medium">{{ $t('account.impersonation.impersonator') }}:</span>
        {{ session?.impersonator_name || '-' }}
      </div>
      <div>
        <span class="font-medium">{{ $t('account.impersonation.session_status') }}:</span>
        {{ $t(`account.impersonation.status_${session?.status}`) || '-' }}
      </div>
    </div>
    <!-- audit log table -->
    <!-- //// check breakpoint, skeleton-columns -->
    <NeTable
      :aria-label="$t('account.impersonation.impersonation_audit_log')"
      card-breakpoint="xl"
      :loading="state.status === 'pending'"
      :skeleton-columns="6"
      :skeleton-rows="7"
    >
      <NeTableHead>
        <NeTableHeadCell>{{ $t('account.impersonation.timestamp') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('account.impersonation.action_type') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('account.impersonation.http_method') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('account.impersonation.api_endpoint') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('account.impersonation.request_data') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('account.impersonation.response_status') }}</NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="(item, index) in sessionsPage" :key="index">
          <NeTableCell :data-label="$t('account.impersonation.timestamp')">
            {{ item.timestamp ? formatDateTime(new Date(item.timestamp), locale) : '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('account.impersonation.action_type')">
            {{ $t(`account.impersonation.action_${item.action_type}`) }}
          </NeTableCell>
          <NeTableCell :data-label="$t('account.impersonation.http_method')">
            {{ item.http_method || '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('account.impersonation.api_endpoint')">
            {{ item.api_endpoint || '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('account.impersonation.request_data')">
            <NeTooltip v-if="item.request_data" placement="left">
              <template #trigger>
                <NeButton kind="tertiary" class="-ml-2.5">
                  {{ $t('account.impersonation.show_request_data') }}
                </NeButton>
              </template>
              <template #content>
                <div class="flex flex-col items-start gap-2">
                  <pre>{{ prettyPrintJsonString(item.request_data, true) || '-' }}</pre>
                  <NeLink invertedTheme @click="copyRequestDataToClipboard(item.request_data)">
                    {{
                      requestDataJustCopied
                        ? t('common.copied')
                        : t('account.impersonation.copy_request_data')
                    }}
                  </NeLink>
                </div>
              </template>
            </NeTooltip>
            <span v-else>-</span>
          </NeTableCell>
          <NeTableCell :data-label="$t('account.impersonation.response_status')">
            {{ item.response_status || '-' }}
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
          class="z-120"
          @select-page="
            (page: number) => {
              pageNum = page
            }
          "
          @select-page-size="
            (size: number) => {
              pageSize = size
              savePageSizeToStorage(SESSION_AUDIT_TABLE_ID, size)
            }
          "
        />
      </template>
    </NeTable>
  </NeModal>
</template>
