<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeHeading } from '@nethesis/vue-components'
import UsersTable from '@/components/users/UsersTable.vue'
import { computed, ref } from 'vue'
import { faCirclePlus } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { PRODUCT_NAME } from '@/lib/config'
import { canManageUsers } from '@/lib/permissions'
import { useUsers } from '@/queries/users/users'
// import { useI18n } from 'vue-i18n' ////
// import { getExport } from '@/lib/users' ////

// const { t } = useI18n() ////
const {
  state,
  // asyncStatus, ////
  // pageNum,
  // pageSize,
  // textFilter,
  debouncedTextFilter,
  // sortBy, ////
  // sortDescending,
} = useUsers()

const isShownCreateUserDrawer = ref(false)

const usersPage = computed(() => {
  return state.value.data?.users
})

//// TODO wait for backend fix
// function getBulkActionsMenuItems() {
//   return [
//     {
//       id: 'exportFilteredToPdf',
//       label: t('users.export_users_to_pdf'),
//       icon: faFilePdf,
//       // action: () => exportUsers('pdf'), ////
//       disabled: !state.value.data?.users,
//     },
//     {
//       id: 'exportFilteredToCsv',
//       label: t('users.export_users_to_csv'),
//       icon: faFileCsv,
//       // action: () => exportUsers('csv'), ////
//       disabled: !state.value.data?.users,
//     },
//   ]
// }

//// TODO wait for backend fix
// async function exportUsers(format: 'pdf' | 'csv') {
//   try {
//     const exportData = await getExport(
//       format,
//       debouncedTextFilter.value,
//       productFilter.value,
//       createdByFilter.value,
//       versionFilter.value,
//       statusFilter.value,
//       sortBy.value,
//       sortDescending.value,
//     )
//     const fileName = `${t('users.title')}.${format}`
//     downloadFile(exportData, fileName, format)
//   } catch (error) {
//     console.error(`Cannot export users to ${format}:`, error)
//     throw error
//   }
// }
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('users.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('users.page_description', { productName: PRODUCT_NAME }) }}
      </div>
      <!-- create user -->
      <NeButton
        v-if="canManageUsers() && (usersPage?.length || debouncedTextFilter)"
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
    <UsersTable
      :isShownCreateUserDrawer="isShownCreateUserDrawer"
      @close-drawer="isShownCreateUserDrawer = false"
    />
  </div>
</template>
