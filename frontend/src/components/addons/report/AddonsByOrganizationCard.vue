<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Which companies hold the add-ons. Companies run to the hundreds on the real
  fleet, so unlike the rest of the report this slice pages and searches on the
  server: the search box and the paginator both drive the request.

  The design calls this "by organization"; the UI has said "company" since long
  before it, and only the code keeps the API's word.
-->

<script setup lang="ts">
import { faMagnifyingGlass } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeEmptyState,
  NePaginator,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  NeTextInput,
} from '@nethesis/vue-components'
import { computed } from 'vue'
import { ADDONS_REPORT_ORGANIZATIONS_TABLE_ID } from '@/lib/addons/addonsReport'
import { getOrganizationIcon } from '@/lib/organizations/organizations'
import { PAGE_SIZE_OPTIONS, savePageSizeToStorage } from '@/lib/tablePageSize'
import { useAddonReportOrganizations } from '@/queries/addons/addonsReport'
import ReportCard from './ReportCard.vue'

const { state: organizations, pageNum, pageSize, textFilter } = useAddonReportOrganizations()

const rows = computed(() => organizations.value.data?.organizations ?? [])
const total = computed(() => organizations.value.data?.total ?? 0)
const loading = computed(() => organizations.value.status === 'pending')
</script>

<template>
  <ReportCard :title="$t('addons.by_company')">
    <div class="flex flex-col gap-4">
      <NeTextInput
        v-model="textFilter"
        is-search
        :placeholder="$t('addons.filter_companies')"
        class="max-w-48 sm:max-w-xs"
        @blur="textFilter = textFilter.trim()"
      />
      <!-- searching the whole fleet can legitimately match nothing -->
      <NeEmptyState
        v-if="!loading && !rows.length"
        :title="$t('organizations.no_organizations')"
        :description="$t('common.try_changing_search_filters')"
        :icon="faMagnifyingGlass"
        class="bg-white dark:bg-gray-950"
      />
      <NeTable
        v-else
        :aria-label="$t('addons.by_company')"
        card-breakpoint="xl"
        :loading="loading"
        :skeleton-columns="4"
        :skeleton-rows="5"
      >
        <NeTableHead>
          <NeTableHeadCell>{{ $t('organizations.organization') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('systems.title') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('addons.status_active') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('addons.total_addons') }}</NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="row in rows" :key="row.organization_id">
            <NeTableCell :data-label="$t('organizations.organization')">
              <div class="flex items-center gap-2">
                <FontAwesomeIcon
                  :icon="getOrganizationIcon(row.org_type)"
                  class="text-tertiary-neutral size-4 shrink-0"
                  aria-hidden="true"
                />
                <span>{{ row.organization_name }}</span>
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('systems.title')">{{ row.systems }}</NeTableCell>
            <NeTableCell :data-label="$t('addons.status_active')">{{ row.active }}</NeTableCell>
            <NeTableCell :data-label="$t('addons.total_addons')">{{ row.total }}</NeTableCell>
          </NeTableRow>
        </NeTableBody>
        <template #paginator>
          <NePaginator
            :current-page="pageNum"
            :total-rows="total"
            :page-size="pageSize"
            :page-sizes="PAGE_SIZE_OPTIONS"
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
                savePageSizeToStorage(ADDONS_REPORT_ORGANIZATIONS_TABLE_ID, size)
              }
            "
          />
        </template>
      </NeTable>
    </div>
  </ReportCard>
</template>
