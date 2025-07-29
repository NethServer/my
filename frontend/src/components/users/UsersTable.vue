<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { getUsers, searchStringInUser, type User } from '@/lib/users'
import { useLoginStore } from '@/stores/login'
import {
  faCircleInfo,
  faCirclePlus,
  faUserGroup,
  faPenToSquare,
  faTrash,
  faKey,
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeTable,
  NeTableHead,
  NeTableHeadCell,
  NeTableBody,
  NeTableRow,
  NeTableCell,
  NePaginator,
  useItemPagination,
  NeButton,
  NeEmptyState,
  NeInlineNotification,
  NeTextInput,
  NeSpinner,
  NeDropdown,
  useSort,
  type SortEvent,
  NeSortDropdown,
  NeBadge,
  sortByProperty,
} from '@nethesis/vue-components'
import { useQuery } from '@pinia/colada'
import { computed, ref, watch } from 'vue'
import CreateOrEditUserDrawer from './CreateOrEditUserDrawer.vue'
import { useI18n } from 'vue-i18n'
import DeleteUserModal from './DeleteUserModal.vue'
import { loadPageSizeFromStorage, savePageSizeToStorage } from '@/lib/tablePageSize'
import ResetPasswordModal from './ResetPasswordModal.vue'
import PasswordChangedModal from './PasswordChangedModal.vue'

const { isShownCreateUserDrawer = false } = defineProps<{
  isShownCreateUserDrawer: boolean
}>()

const emit = defineEmits(['close-drawer'])

const { t } = useI18n()
const loginStore = useLoginStore()
const { state: users, asyncStatus: usersAsyncStatus } = useQuery({
  key: ['users'],
  enabled: () => !!loginStore.jwtToken,
  query: getUsers,
})

const currentUser = ref<User | undefined>()
const textFilter = ref('')
const isShownCreateOrEditUserDrawer = ref(false)
const isShownDeleteUserModal = ref(false)
const isShownResetPasswordModal = ref(false)
const isShownPasswordChangedModal = ref(false)
const newPassword = ref<string>('')
const tableId = 'usersTable'
const pageSize = ref(10)
const sortKey = ref<keyof User>('name')
const sortDescending = ref(false)

const filteredUsers = computed(() => {
  if (!users.value.data?.length) {
    return []
  }

  if (!textFilter.value.trim()) {
    return users.value.data
  } else {
    return users.value.data.filter((user) => searchStringInUser(textFilter.value, user))
  }
})

const sortFunctions = {
  // custom sorting function for organization attribute
  organization: (a: User, b: User) => {
    if ((!a.organization || !a.organization.name) && (!b.organization || !b.organization.name))
      return 0
    if (!a.organization || !a.organization.name) return 1
    if (!b.organization || !b.organization.name) return -1
    return a.organization.name.localeCompare(b.organization.name || '')
  },
}

const { sortedItems } = useSort(filteredUsers, sortKey, sortDescending, sortFunctions)

const { currentPage, paginatedItems } = useItemPagination(() => sortedItems.value, {
  itemsPerPage: pageSize,
})

watch(
  () => isShownCreateUserDrawer,
  () => {
    if (isShownCreateUserDrawer) {
      showCreateUserDrawer()
    }
  },
  { immediate: true },
)

watch(
  () => loginStore.userInfo?.email,
  (email) => {
    if (email) {
      pageSize.value = loadPageSizeFromStorage(tableId)
    }
  },
  { immediate: true },
)

function clearFilters() {
  textFilter.value = ''
}

function showCreateUserDrawer() {
  currentUser.value = undefined
  isShownCreateOrEditUserDrawer.value = true
}

function showEditUserDrawer(user: User) {
  currentUser.value = user
  isShownCreateOrEditUserDrawer.value = true
}

function showDeleteUserModal(user: User) {
  currentUser.value = user
  isShownDeleteUserModal.value = true
}

function showResetPasswordModal(user: User) {
  currentUser.value = user
  isShownResetPasswordModal.value = true
}

function onPasswordChanged(newPwd: string) {
  newPassword.value = newPwd
  isShownPasswordChangedModal.value = true
}

function onCloseDrawer() {
  isShownCreateOrEditUserDrawer.value = false
  emit('close-drawer')
}

function getKebabMenuItems(user: User) {
  return [
    {
      id: 'resetPassword',
      label: t('users.reset_password'),
      icon: faKey,
      action: () => showResetPasswordModal(user),
      disabled: usersAsyncStatus.value === 'loading',
    },
    {
      id: 'deleteAccount',
      label: t('common.delete'),
      icon: faTrash,
      danger: true,
      action: () => showDeleteUserModal(user),
      disabled: usersAsyncStatus.value === 'loading',
    },
  ]
}

const onSort = (payload: SortEvent) => {
  sortKey.value = payload.key as keyof User
  sortDescending.value = payload.descending
}

const onClosePasswordChangedModal = () => {
  isShownPasswordChangedModal.value = false
  newPassword.value = ''
}
</script>

<template>
  <div>
    <!-- get users error notification -->
    <NeInlineNotification
      v-if="users.status === 'error'"
      kind="error"
      :title="$t('users.cannot_retrieve_users')"
      :description="users.error.message"
      class="mb-6"
    />
    <!-- table toolbar -->
    <div class="mb-6 flex items-center gap-4">
      <div class="flex w-full items-center justify-between gap-4">
        <!-- filters -->
        <div class="flex flex-wrap items-center gap-4">
          <!-- text filter -->
          <NeTextInput
            v-model.trim="textFilter"
            is-search
            :placeholder="$t('users.filter_users')"
            class="max-w-48 sm:max-w-sm"
          />
          <NeSortDropdown
            v-model:sort-key="sortKey"
            v-model:sort-descending="sortDescending"
            :label="t('sort.sort')"
            :options="[
              { id: 'name', label: t('users.name') },
              { id: 'email', label: t('users.email') },
              { id: 'organization', label: t('users.organization') },
            ]"
            :open-menu-aria-label="t('ne_dropdown.open_menu')"
            :sort-by-label="t('sort.sort_by')"
            :sort-direction-label="t('sort.direction')"
            :ascending-label="t('sort.ascending')"
            :descending-label="t('sort.descending')"
            class="xl:hidden"
          />
        </div>
        <!-- update indicator -->
        <div
          v-if="usersAsyncStatus === 'loading' && users.status !== 'pending'"
          class="flex items-center gap-2"
        >
          <NeSpinner color="white" />
          <div class="text-gray-500 dark:text-gray-400">
            {{ $t('common.updating') }}
          </div>
        </div>
      </div>
    </div>
    <!-- //// check breakpoint, skeleton-columns -->
    <NeTable
      :sort-key="sortKey"
      :sort-descending="sortDescending"
      :aria-label="$t('users.title')"
      card-breakpoint="xl"
      :loading="users.status === 'pending'"
      :skeleton-columns="5"
      :skeleton-rows="7"
    >
      <NeTableHead>
        <NeTableHeadCell sortable column-key="name" @sort="onSort">{{
          $t('users.name')
        }}</NeTableHeadCell>
        <NeTableHeadCell sortable column-key="email" @sort="onSort">{{
          $t('users.email')
        }}</NeTableHeadCell>
        <NeTableHeadCell sortable column-key="organization" @sort="onSort">{{
          $t('users.organization')
        }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('users.roles') }}</NeTableHeadCell>
        <NeTableHeadCell>
          <!-- no header for actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <!-- empty state -->
        <NeTableRow v-if="!users.data?.length">
          <NeTableCell colspan="5">
            <NeEmptyState
              :title="$t('users.no_user')"
              :icon="faUserGroup"
              class="bg-white dark:bg-gray-950"
            >
              <!-- create user -->
              <NeButton kind="secondary" size="lg" class="shrink-0" @click="showCreateUserDrawer()">
                <template #prefix>
                  <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
                </template>
                {{ $t('users.create_user') }}
              </NeButton>
            </NeEmptyState>
          </NeTableCell>
        </NeTableRow>
        <!-- no user matching filter -->
        <NeTableRow v-else-if="!filteredUsers.length">
          <NeTableCell colspan="4">
            <NeEmptyState
              :title="$t('users.no_user_found')"
              :description="$t('common.try_changing_search_filters')"
              :icon="faCircleInfo"
              class="bg-white dark:bg-gray-950"
            >
              <NeButton kind="tertiary" @click="clearFilters">
                {{ $t('common.clear_filters') }}</NeButton
              >
            </NeEmptyState>
          </NeTableCell>
        </NeTableRow>
        <NeTableRow v-for="(item, index) in paginatedItems" v-else :key="index">
          <NeTableCell :data-label="$t('users.name')">
            {{ item.name }}
          </NeTableCell>
          <NeTableCell :data-label="$t('users.email')" class="break-all xl:break-normal">
            {{ item.email }}
          </NeTableCell>
          <NeTableCell :data-label="$t('users.organization')">
            {{ item.organization?.name || '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('users.roles')">
            <div class="flex flex-wrap gap-1">
              <NeBadge
                v-for="role in item.roles?.sort(sortByProperty('name'))"
                :key="role.id"
                :text="role.name"
                kind="custom"
                customColorClasses="bg-indigo-100 text-indigo-800 dark:bg-indigo-700 dark:text-indigo-100"
                class="inline-block"
              ></NeBadge>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('common.actions')">
            <div class="-ml-2.5 flex gap-2 xl:ml-0 xl:justify-end">
              <NeButton
                kind="tertiary"
                @click="showEditUserDrawer(item)"
                :disabled="usersAsyncStatus === 'loading'"
              >
                <template #prefix>
                  <FontAwesomeIcon :icon="faPenToSquare" class="h-4 w-4" aria-hidden="true" />
                </template>
                {{ $t('common.edit') }}
              </NeButton>
              <!-- kebab menu -->
              <NeDropdown :items="getKebabMenuItems(item)" :align-to-right="true" />
            </div>
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
      <template #paginator>
        <NePaginator
          :current-page="currentPage"
          :total-rows="sortedItems.length"
          :page-size="pageSize"
          :nav-pagination-label="$t('ne_table.pagination')"
          :next-label="$t('ne_table.go_to_next_page')"
          :previous-label="$t('ne_table.go_to_previous_page')"
          :range-of-total-label="$t('ne_table.of')"
          :page-size-label="$t('ne_table.show')"
          @select-page="
            (page: number) => {
              currentPage = page
            }
          "
          @select-page-size="
            (size: number) => {
              pageSize = size
              savePageSizeToStorage(tableId, size)
            }
          "
        />
      </template>
    </NeTable>
    <!-- side drawer -->
    <CreateOrEditUserDrawer
      :is-shown="isShownCreateOrEditUserDrawer"
      :current-user="currentUser"
      @close="onCloseDrawer"
    />
    <!-- delete user modal -->
    <DeleteUserModal
      :visible="isShownDeleteUserModal"
      :user="currentUser"
      @close="isShownDeleteUserModal = false"
    />
    <!-- reset password modal -->
    <ResetPasswordModal
      :visible="isShownResetPasswordModal"
      :user="currentUser"
      @close="isShownResetPasswordModal = false"
      @password-changed="onPasswordChanged"
    />
    <!-- password changed modal -->
    <PasswordChangedModal
      :visible="isShownPasswordChangedModal"
      :user="currentUser"
      :new-password="newPassword"
      @close="onClosePasswordChangedModal"
    />
  </div>
</template>
