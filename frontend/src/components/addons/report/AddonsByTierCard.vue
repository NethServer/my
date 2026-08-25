<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  How the grants split across the shop tiers of each add-on — the device or
  mailbox bands the product is sold in. One add-on contributes as many rows as
  it has tiers, so this slice pages and searches on the server like the
  organizations one.
-->

<script setup lang="ts">
import { faMagnifyingGlass } from '@fortawesome/free-solid-svg-icons'
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
import { ADDONS_REPORT_TIERS_TABLE_ID } from '@/lib/addons/addonsReport'
import { PAGE_SIZE_OPTIONS, savePageSizeToStorage } from '@/lib/tablePageSize'
import { useAddonReportTiers } from '@/queries/addons/addonsReport'
import ReportCard from './ReportCard.vue'

const { displayNames } = defineProps<{
  // catalog id -> display name: the endpoint returns tiers by add-on id
  displayNames: Record<string, string>
}>()

const { state: tiers, pageNum, pageSize, textFilter } = useAddonReportTiers()

const rows = computed(() => tiers.value.data?.tiers ?? [])
const total = computed(() => tiers.value.data?.total ?? 0)
const loading = computed(() => tiers.value.status === 'pending')
</script>

<template>
  <ReportCard :title="$t('addons.by_tier')">
    <div class="flex flex-col gap-4">
      <NeTextInput
        v-model="textFilter"
        is-search
        :placeholder="$t('addons.filter_tier')"
        class="max-w-48 sm:max-w-xs"
        @blur="textFilter = textFilter.trim()"
      />
      <!-- an add-on sold as a single product has no tiers at all, so this card
           is empty on a fleet that never bought a tiered one -->
      <NeEmptyState
        v-if="!loading && !rows.length"
        :title="$t('addons.no_tiers')"
        :description="$t('addons.no_tiers_description')"
        :icon="faMagnifyingGlass"
        class="bg-white dark:bg-gray-950"
      />
      <NeTable
        v-else
        :aria-label="$t('addons.by_tier')"
        card-breakpoint="xl"
        :loading="loading"
        :skeleton-columns="3"
        :skeleton-rows="5"
      >
        <NeTableHead>
          <NeTableHeadCell>{{ $t('addons.addon') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('addons.tier') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('addons.count') }}</NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="row in rows" :key="`${row.entitlement} ${row.label}`">
            <NeTableCell :data-label="$t('addons.addon')">
              {{ displayNames[row.entitlement] ?? row.entitlement }}
            </NeTableCell>
            <NeTableCell :data-label="$t('addons.tier')">{{ row.label }}</NeTableCell>
            <NeTableCell :data-label="$t('addons.count')">{{ row.count }}</NeTableCell>
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
                savePageSizeToStorage(ADDONS_REPORT_TIERS_TABLE_ID, size)
              }
            "
          />
        </template>
      </NeTable>
    </div>
  </ReportCard>
</template>
