<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeDropdown, NeHeading } from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import {
  faChevronDown,
  faCirclePlus,
  faFileCsv,
  faFilePdf,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { canManageSystems } from '@/lib/permissions'
import SystemsTable from '@/components/systems/SystemsTable.vue'
import { useSystems } from '@/queries/systems/systems'
import { useI18n } from 'vue-i18n'
import { getExport } from '@/lib/systems/systems'
import { downloadFile } from '@/lib/common'

const { t } = useI18n()

const {
  state,
  debouncedTextFilter,
  productFilter,
  createdByFilter,
  versionFilter,
  statusFilter,
  sortBy,
  sortDescending,
} = useSystems()

const isShownCreateSystemDrawer = ref(false)

const systemsPage = computed(() => {
  return state.value.data?.systems
})

function getBulkActionsMenuItems() {
  return [
    {
      id: 'exportFilteredToPdf',
      label: t('systems.export_systems_to_pdf'),
      icon: faFilePdf,
      action: () => exportSystems('pdf'),
      disabled: !state.value.data?.systems,
    },
    {
      id: 'exportFilteredToCsv',
      label: t('systems.export_systems_to_csv'),
      icon: faFileCsv,
      action: () => exportSystems('csv'),
      disabled: !state.value.data?.systems,
    },
  ]
}

async function exportSystems(format: 'pdf' | 'csv') {
  try {
    const exportData = await getExport(
      format,
      undefined,
      debouncedTextFilter.value,
      productFilter.value,
      createdByFilter.value,
      versionFilter.value,
      statusFilter.value,
      sortBy.value,
      sortDescending.value,
    )
    const fileName = `${t('systems.title')}.${format}`
    downloadFile(exportData, fileName, format)
  } catch (error) {
    console.error(`Cannot export systems to ${format}:`, error)
    throw error
  }
}
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('systems.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('systems.page_description') }}
      </div>
      <div class="flex flex-row-reverse items-center gap-4 xl:flex-row">
        <NeDropdown
          :items="getBulkActionsMenuItems()"
          align-to-right
          :openMenuAriaLabel="$t('ne_dropdown.open_menu')"
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
        <!-- create system -->
        <NeButton
          v-if="canManageSystems()"
          kind="primary"
          size="lg"
          class="shrink-0"
          @click="isShownCreateSystemDrawer = true"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
          </template>
          {{ $t('systems.create_system') }}
        </NeButton>
      </div>
    </div>
    <SystemsTable
      :isShownCreateSystemDrawer="isShownCreateSystemDrawer"
      @close-drawer="isShownCreateSystemDrawer = false"
    />
  </div>
</template>
