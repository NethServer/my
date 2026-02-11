<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeDropdown, NeHeading } from '@nethesis/vue-components'
import DistributorsTable from '@/components/distributors/DistributorsTable.vue'
import { computed, ref } from 'vue'
import {
  faChevronDown,
  faCirclePlus,
  faFileCsv,
  faFilePdf,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { canManageDistributors, canReadDistributors } from '@/lib/permissions'
import { useDistributors } from '@/queries/organizations/distributors'
import { useI18n } from 'vue-i18n'
// import { downloadFile } from '@/lib/common' ////

const { t } = useI18n()
const { state, debouncedTextFilter } = useDistributors()

const isShownCreateDistributorDrawer = ref(false)

const distributorsPage = computed(() => {
  return state.value.data?.distributors
})

function getBulkActionsMenuItems() {
  return [
    {
      id: 'exportFilteredToPdf',
      label: t('distributors.export_distributors_to_pdf'),
      icon: faFilePdf,
      // action: () => exportDistributors('pdf'), ////
      disabled: !state.value.data?.distributors,
    },
    {
      id: 'exportFilteredToCsv',
      label: t('distributors.export_distributors_to_csv'),
      icon: faFileCsv,
      // action: () => exportDistributors('csv'), ////
      disabled: !state.value.data?.distributors,
    },
  ]
}

//// implement after filters fix on backend
// async function exportDistributors(format: 'pdf' | 'csv') {
//   try {
//     const exportData = await getExport(
//       format,
//       undefined,
//       debouncedTextFilter.value,
//       productFilter.value,
//       createdByFilter.value,
//       versionFilter.value,
//       statusFilter.value,
//       sortBy.value,
//       sortDescending.value,
//     )
//     const fileName = `${t('distributors.title')}.${format}`
//     downloadFile(exportData, fileName, format)
//   } catch (error) {
//     console.error(`Cannot export distributors to ${format}:`, error)
//     throw error
//   }
// }
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('distributors.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('distributors.page_description') }}
      </div>
      <!-- v-if condition is the opposite of empty state condition in DistributorsTable.vue -->
      <div
        v-if="!(state.status === 'success' && !distributorsPage?.length && !debouncedTextFilter)"
        class="flex items-center gap-4"
      >
        <NeDropdown
          :items="getBulkActionsMenuItems()"
          align-to-right
          :openMenuAriaLabel="$t('ne_dropdown.open_menu')"
          v-if="canReadDistributors()"
        >
          >
          <template #button>
            <NeButton>
              <template #suffix>
                <FontAwesomeIcon
                  :icon="faChevronDown"
                  class="h-4 w-4"
                  aria-hidden="true"
                /> </template
              >{{ $t('common.actions') }}</NeButton
            >
          </template>
        </NeDropdown>
        <!-- create distributor -->
        <NeButton
          v-if="canManageDistributors() && (distributorsPage?.length || debouncedTextFilter)"
          kind="primary"
          size="lg"
          class="shrink-0"
          @click="isShownCreateDistributorDrawer = true"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
          </template>
          {{ $t('distributors.create_distributor') }}
        </NeButton>
      </div>
    </div>
    <DistributorsTable
      :isShownCreateDistributorDrawer="isShownCreateDistributorDrawer"
      @close-drawer="isShownCreateDistributorDrawer = false"
    />
  </div>
</template>
