<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeDropdown, NeHeading } from '@nethesis/vue-components'
import UsersTable from '@/components/users/UsersTable.vue'
import { computed, ref } from 'vue'
import {
  faChevronDown,
  faCirclePlus,
  faFileCsv,
  faFilePdf,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { PRODUCT_NAME } from '@/lib/config'
import { canManageUsers, canReadUsers } from '@/lib/permissions'
import { useUsers } from '@/queries/users/users'
import { useI18n } from 'vue-i18n'
import { getExport } from '@/lib/users/users'
import { downloadFile } from '@/lib/common'

const { t } = useI18n()
const {
  state,
  debouncedTextFilter,
  organizationFilter,
  roleFilter,
  statusFilter,
  sortBy,
  sortDescending,
  areDefaultFiltersApplied,
} = useUsers()

const isShownCreateUserDrawer = ref(false)

const usersPage = computed(() => {
  return state.value.data?.users
})

function getBulkActionsMenuItems() {
  return [
    {
      id: 'exportFilteredToPdf',
      label: t('users.export_users_to_pdf'),
      icon: faFilePdf,
      action: () => exportUsers('pdf'),
      disabled: !state.value.data?.users,
    },
    {
      id: 'exportFilteredToCsv',
      label: t('users.export_users_to_csv'),
      icon: faFileCsv,
      action: () => exportUsers('csv'),
      disabled: !state.value.data?.users,
    },
  ]
}

async function exportUsers(format: 'pdf' | 'csv') {
  try {
    const exportData = await getExport(
      format,
      debouncedTextFilter.value,
      organizationFilter.value,
      roleFilter.value,
      statusFilter.value,
      sortBy.value,
      sortDescending.value,
    )
    const fileName = `${t('users.title')}.${format}`
    downloadFile(exportData, fileName, format)
  } catch (error) {
    console.error(`Cannot export users to ${format}:`, error)
    throw error
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
      <div
        v-if="!(state.status === 'success' && !usersPage?.length && areDefaultFiltersApplied)"
        class="flex items-center gap-4"
      >
        <NeDropdown
          :items="getBulkActionsMenuItems()"
          align-to-right
          :openMenuAriaLabel="$t('ne_dropdown.open_menu')"
          v-if="canReadUsers()"
        >
          <template #button>
            <NeButton>
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
  </div>
</template>
