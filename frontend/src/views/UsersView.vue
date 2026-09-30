<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeDropdown, NeHeading } from '@nethesis/vue-components'
import UsersTable from '@/components/users/UsersTable.vue'
import ImportUsersModal from '@/components/users/ImportUsersModal.vue'
import { ref } from 'vue'
import {
  faChevronDown,
  faCircleArrowUp,
  faCirclePlus,
  faFileCsv,
  faFilePdf,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { PRODUCT_NAME } from '@/lib/config'
import { canManageUsers } from '@/lib/permissions'
import { useUsers } from '@/queries/users/users'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { getExport, type UserStatus } from '@/lib/users/users'
import { downloadFile, exportFileName, getExportLimitError } from '@/lib/common'

const { t } = useI18n()
const notificationsStore = useNotificationsStore()

// the export runs on the whole filtered list and can take a while: one at a time,
// with the Actions button showing progress
const isExporting = ref(false)
const {
  state,
  debouncedTextFilter,
  organizationFilter,
  roleFilter,
  statusFilter,
  sortBy,
  sortDescending,
} = useUsers()

const isShownCreateUserDrawer = ref(false)
const isShownImportUsersModal = ref(false)

function getBulkActionsMenuItems() {
  return [
    ...(canManageUsers()
      ? [
          {
            id: 'importUsers',
            label: t('import.users.import_users'),
            icon: faCircleArrowUp,
            action: () => (isShownImportUsersModal.value = true),
          },
        ]
      : []),
    {
      id: 'exportFilteredToPdf',
      label: t('users.export_users_to_pdf'),
      icon: faFilePdf,
      action: () => exportUsers('pdf'),
      disabled: isExporting.value || !state.value.data?.users.length,
    },
    {
      id: 'exportFilteredToCsv',
      label: t('users.export_users_to_csv'),
      icon: faFileCsv,
      action: () => exportUsers('csv'),
      disabled: isExporting.value || !state.value.data?.users.length,
    },
  ]
}

async function exportUsers(format: 'pdf' | 'csv') {
  try {
    isExporting.value = true
    const exportData = await getExport(
      format,
      debouncedTextFilter.value,
      organizationFilter.value.map((o) => o.id),
      roleFilter.value.map((o) => o.id),
      statusFilter.value.map((o) => o.id) as UserStatus[],
      [], // created by: not filtered from the tables
      sortBy.value,
      sortDescending.value,
    )
    const fileName = exportFileName(t('users.title'), format)
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
    console.error(`Cannot export users to ${format}:`, error)
    throw error
  } finally {
    isExporting.value = false
  }
}
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('users.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('users.page_description', { productName: PRODUCT_NAME }) }}
      </div>
      <div class="flex items-center gap-4">
        <NeDropdown
          :items="getBulkActionsMenuItems()"
          align-to-right
          :openMenuAriaLabel="$t('ne_dropdown.open_menu')"
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
        <!-- create user -->
        <NeButton
          v-if="canManageUsers()"
          kind="primary"
          size="lg"
          class="shrink-0"
          @click="isShownCreateUserDrawer = true"
        >
          <template #prefix>
            <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
          </template>
          {{ $t('users.create_user') }}
        </NeButton>
      </div>
    </div>
    <UsersTable
      :isShownCreateUserDrawer="isShownCreateUserDrawer"
      @close-drawer="isShownCreateUserDrawer = false"
    />
    <!-- import users drawer -->
    <ImportUsersModal
      :is-shown="isShownImportUsersModal"
      @close="isShownImportUsersModal = false"
    />
  </div>
</template>
