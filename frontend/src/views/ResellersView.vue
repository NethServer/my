<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeDropdown, NeHeading } from '@nethesis/vue-components'
import ResellersTable from '@/components/resellers/ResellersTable.vue'
import { ref } from 'vue'
import {
  faChevronDown,
  faCirclePlus,
  faFileCsv,
  faFilePdf,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { canManageResellers } from '@/lib/permissions'
import { useResellers } from '@/queries/organizations/resellers'
import { useI18n } from 'vue-i18n'
import { getExport } from '@/lib/organizations/resellers'
import { downloadFile } from '@/lib/common'

const { t } = useI18n()
const { state, debouncedTextFilter, statusFilter, sortBy, sortDescending } = useResellers()

const isShownCreateResellerDrawer = ref(false)

function getBulkActionsMenuItems() {
  return [
    {
      id: 'exportFilteredToPdf',
      label: t('resellers.export_resellers_to_pdf'),
      icon: faFilePdf,
      action: () => exportResellers('pdf'),
      disabled: !state.value.data?.resellers,
    },
    {
      id: 'exportFilteredToCsv',
      label: t('resellers.export_resellers_to_csv'),
      icon: faFileCsv,
      action: () => exportResellers('csv'),
      disabled: !state.value.data?.resellers,
    },
  ]
}

async function exportResellers(format: 'pdf' | 'csv') {
  try {
    const exportData = await getExport(
      format,
      debouncedTextFilter.value,
      statusFilter.value,
      sortBy.value,
      sortDescending.value,
    )
    const fileName = `${t('resellers.title')}.${format}`
    downloadFile(exportData, fileName, format)
  } catch (error) {
    console.error(`Cannot export resellers to ${format}:`, error)
    throw error
  }
}
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('resellers.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('resellers.page_description') }}
      </div>
      <div class="flex flex-row-reverse items-center gap-4 xl:flex-row">
        <NeDropdown
          :items="getBulkActionsMenuItems()"
          align-to-right
          :openMenuAriaLabel="$t('ne_dropdown.open_menu')"
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
        <!-- create reseller -->
        <NeButton
          v-if="canManageResellers()"
          kind="primary"
          size="lg"
          class="shrink-0"
          @click="isShownCreateResellerDrawer = true"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
          </template>
          {{ $t('resellers.create_reseller') }}
        </NeButton>
      </div>
    </div>
    <ResellersTable
      :isShownCreateResellerDrawer="isShownCreateResellerDrawer"
      @close-drawer="isShownCreateResellerDrawer = false"
    />
  </div>
</template>
