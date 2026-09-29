<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeDropdown, NeHeading } from '@nethesis/vue-components'
import { ref } from 'vue'
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
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { getExport, type SystemStatus } from '@/lib/systems/systems'
import { downloadFile, exportFileName, getExportLimitError } from '@/lib/common'

const { t } = useI18n()
const notificationsStore = useNotificationsStore()

// the export runs on the whole filtered list and can take a while: one at a time,
// with the Actions button showing progress
const isExporting = ref(false)
const route = useRoute()
const router = useRouter()

const {
  state,
  debouncedTextFilter,
  productFilter,
  createdByFilter,
  versionFilter,
  statusFilter,
  organizationFilter,
  addonFilter,
  includeHierarchy,
  sortBy,
  sortDescending,
  applyHierarchyFilter,
  resetFilters,
} = useSystems()

// apply the filters requested via query params, then clean the URL
const {
  organization_id: orgId,
  organization_name: orgName,
  include_hierarchy: includeHierarchyParam,
  status,
} = route.query

if (typeof orgId === 'string' && orgId && typeof orgName === 'string' && orgName) {
  if (includeHierarchyParam === 'true') {
    applyHierarchyFilter({ id: orgId, label: orgName })
  } else {
    resetFilters()
    organizationFilter.value = [{ id: orgId, label: orgName }]
  }
  router.replace({ query: {} })
}

if (typeof status === 'string' && ['active', 'inactive', 'unknown', 'suspended'].includes(status)) {
  resetFilters()
  statusFilter.value = [{ id: status, label: status }]
  router.replace({ query: {} })
}

const isShownCreateSystemDrawer = ref(false)

function getBulkActionsMenuItems() {
  return [
    {
      id: 'exportFilteredToPdf',
      label: t('systems.export_systems_to_pdf'),
      icon: faFilePdf,
      action: () => exportSystems('pdf'),
      disabled: isExporting.value || !state.value.data?.systems.length,
    },
    {
      id: 'exportFilteredToCsv',
      label: t('systems.export_systems_to_csv'),
      icon: faFileCsv,
      action: () => exportSystems('csv'),
      disabled: isExporting.value || !state.value.data?.systems.length,
    },
  ]
}

async function exportSystems(format: 'pdf' | 'csv') {
  try {
    isExporting.value = true
    const exportData = await getExport(
      format,
      undefined,
      debouncedTextFilter.value,
      productFilter.value.map((o) => o.id),
      createdByFilter.value.map((o) => o.id),
      versionFilter.value.map((o) => o.id),
      statusFilter.value.map((o) => o.id) as SystemStatus[],
      organizationFilter.value.map((o) => o.id),
      addonFilter.value.map((o) => o.id),
      includeHierarchy.value,
      sortBy.value,
      sortDescending.value,
    )
    const fileName = exportFileName(t('systems.title'), format)
    downloadFile(exportData, fileName, format)
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
    console.error(`Cannot export systems to ${format}:`, error)
    throw error
  } finally {
    isExporting.value = false
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
            <NeButton :disabled="isExporting" :loading="isExporting" loading-position="suffix">
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
