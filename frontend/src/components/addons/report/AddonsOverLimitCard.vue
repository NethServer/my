<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The add-ons in force past the limit of their tier — devices on a firewall,
  instances on a cluster — as collect measured them at the last inventory.
  Nothing is blocked: this is the list Nethesis sales follow up, widest
  overrun first, and only Nethesis is shown it.
-->

<script setup lang="ts">
import {
  faChevronDown,
  faCircleCheck,
  faFileCsv,
  faFilePdf,
  faTriangleExclamation,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeButton,
  NeDropdown,
  NeEmptyState,
  NePaginator,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  NeTextInput,
  NeTooltip,
} from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ADDONS_OVER_LIMIT_TABLE_ID,
  getAddonsOverLimitExport,
  type AddonOverLimit,
} from '@/lib/addons/addonsReport'
import { downloadFile, exportFileName, getExportLimitError } from '@/lib/common'
import { getTierLabel } from '@/lib/addons/systemAddons'
import { PAGE_SIZE_OPTIONS, savePageSizeToStorage } from '@/lib/tablePageSize'
import { useAddons } from '@/queries/addons/addons'
import { useAddonsOverLimit } from '@/queries/addons/addonsReport'
import { useNotificationsStore } from '@/stores/notifications'
import AddonLogo from '../AddonLogo.vue'
import ReportCard from './ReportCard.vue'

const {
  state: overLimit,
  pageNum,
  pageSize,
  textFilter,
  debouncedTextFilter,
} = useAddonsOverLimit()

const rows = computed(() => overLimit.value.data?.grants ?? [])
const total = computed(() => overLimit.value.data?.total ?? 0)
const loading = computed(() => overLimit.value.status === 'pending')

// the rows name the add-on by id: its name and what it is measured in come
// from the catalog
const { state: catalog } = useAddons()
const addonOf = (id: string) => catalog.value.data?.find((item) => item.id === id)

const { t } = useI18n()
const notificationsStore = useNotificationsStore()

// the whole list, not just the page on screen, with the search applied: one
// export at a time, the Actions button showing progress
const isExporting = ref(false)

const exportItems = computed(() => [
  {
    id: 'exportToPdf',
    label: t('addons.export_to_pdf'),
    icon: faFilePdf,
    action: () => exportOverLimit('pdf'),
    disabled: isExporting.value || !rows.value.length,
  },
  {
    id: 'exportToCsv',
    label: t('addons.export_to_csv'),
    icon: faFileCsv,
    action: () => exportOverLimit('csv'),
    disabled: isExporting.value || !rows.value.length,
  },
])

async function exportOverLimit(format: 'pdf' | 'csv') {
  try {
    isExporting.value = true
    const data = await getAddonsOverLimitExport(format, debouncedTextFilter.value)
    downloadFile(data, exportFileName(t('addons.over_limit'), format), format)
  } catch (error) {
    const limit = getExportLimitError(error)
    if (limit) {
      notificationsStore.createNotification({
        kind: 'error',
        title: t('common.too_many_records_to_export'),
        description: t('common.too_many_records_to_export_description', limit),
      })
      return
    }
    console.error(`Cannot export add-ons over the tier to ${format}:`, error)
    throw error
  } finally {
    isExporting.value = false
  }
}

// both the system and the add-on lead to the add-ons of the system
const systemAddonsRoute = (row: AddonOverLimit) => ({
  name: 'system_detail',
  params: { systemId: row.system_id },
  query: { tab: 'addons' },
})

const overTierText = (row: AddonOverLimit) => {
  const over = row.usage - row.seats
  const metric = addonOf(row.entitlement)?.usage_metric
  return metric ? t(`addons.over_tier_${metric}`, over) : t('addons.over_tier', { count: over })
}
</script>

<template>
  <ReportCard :title="$t('addons.over_limit')">
    <div class="flex flex-col gap-4">
      <div class="flex items-center justify-between gap-4">
        <NeTextInput
          v-model="textFilter"
          is-search
          :placeholder="$t('addons.filter_over_limit')"
          class="max-w-48 sm:max-w-xs"
          @blur="textFilter = textFilter.trim()"
        />
        <NeDropdown
          :items="exportItems"
          align-to-right
          :open-menu-aria-label="$t('ne_dropdown.open_menu')"
        >
          <template #button>
            <NeButton :disabled="isExporting" :loading="isExporting" loading-position="suffix">
              <template #suffix>
                <FontAwesomeIcon :icon="faChevronDown" class="h-4 w-4" aria-hidden="true" />
              </template>
              {{ $t('common.actions') }}
            </NeButton>
          </template>
        </NeDropdown>
      </div>
      <NeEmptyState
        v-if="!loading && !rows.length"
        :title="$t('addons.no_over_limit')"
        :icon="faCircleCheck"
        class="bg-white dark:bg-gray-950"
      />
      <NeTable
        v-else
        :aria-label="$t('addons.over_limit')"
        card-breakpoint="xl"
        :loading="loading"
        :skeleton-columns="4"
        :skeleton-rows="5"
      >
        <NeTableHead>
          <NeTableHeadCell>{{ $t('addons.system_column') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('addons.addon') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('addons.tier') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('addons.usage_column') }}</NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="row in rows" :key="row.id">
            <NeTableCell :data-label="$t('addons.system_column')">
              <div class="space-y-0.5">
                <router-link
                  :to="systemAddonsRoute(row)"
                  class="cursor-pointer font-medium hover:underline"
                >
                  {{ row.system_name }}
                </router-link>
                <div class="text-tertiary-neutral">{{ row.organization_name }}</div>
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('addons.addon')">
              <div class="flex items-center gap-2">
                <AddonLogo :addon-id="row.entitlement" />
                <router-link
                  :to="systemAddonsRoute(row)"
                  class="cursor-pointer font-medium hover:underline"
                >
                  {{ addonOf(row.entitlement)?.display_name ?? row.entitlement }}
                </router-link>
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('addons.tier')">
              {{ row.variant ? getTierLabel(row.variant, $t, $te) : row.seats }}
            </NeTableCell>
            <NeTableCell :data-label="$t('addons.usage_column')">
              <!-- the tier column already says what is counted: the cell stays
                   short enough for the narrow card -->
              <div class="flex items-center gap-2 whitespace-nowrap">
                <NeTooltip trigger-event="mouseenter focus" placement="top">
                  <template #trigger>
                    <FontAwesomeIcon
                      :icon="faTriangleExclamation"
                      class="size-4 text-amber-700 dark:text-amber-500"
                      aria-hidden="true"
                    />
                    <span class="sr-only">{{ overTierText(row) }}</span>
                  </template>
                  <template #content>{{ overTierText(row) }}</template>
                </NeTooltip>
                <span>{{ row.usage }}/{{ row.seats }}</span>
              </div>
            </NeTableCell>
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
                savePageSizeToStorage(ADDONS_OVER_LIMIT_TABLE_ID, size)
              }
            "
          />
        </template>
      </NeTable>
    </div>
  </ReportCard>
</template>
