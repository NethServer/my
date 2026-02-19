<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { USERS_TABLE_ID, type User } from '@/lib/users/users'
import {
  faCircleInfo,
  faUserGroup,
  faPenToSquare,
  faBoxArchive,
  faKey,
  faUserSecret,
  faCirclePause,
  faCirclePlay,
  faCircleCheck,
  faRotateLeft,
  faBomb,
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
  NeButton,
  NeEmptyState,
  NeInlineNotification,
  NeTextInput,
  NeSpinner,
  NeDropdown,
  type SortEvent,
  NeSortDropdown,
  sortByProperty,
  type NeDropdownItem,
  NeTooltip,
  NeDropdownFilter,
  type FilterOption,
} from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import CreateOrEditUserDrawer from './CreateOrEditUserDrawer.vue'
import { useI18n } from 'vue-i18n'
import DeleteUserModal from './DeleteUserModal.vue'
import DestroyUserModal from './DestroyUserModal.vue'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import ResetPasswordModal from './ResetPasswordModal.vue'
import PasswordChangedModal from './PasswordChangedModal.vue'
import { useUsers } from '@/queries/users/users'
import { canManageUsers, canImpersonateUsers, canDestroyUsers } from '@/lib/permissions'
import { useLoginStore } from '@/stores/login'
import ImpersonateUserModal from './ImpersonateUserModal.vue'
import SuspendUserModal from './SuspendUserModal.vue'
import ReactivateUserModal from './ReactivateUserModal.vue'
import RestoreUserModal from './RestoreUserModal.vue'
import OrganizationIcon from '../organizations/OrganizationIcon.vue'
import UserRoleBadge from './UserRoleBadge.vue'
import { useOrganizationFilter } from '@/queries/users/organizationFilter'
import { userRolesQuery } from '@/queries/users/userRoles'
import { useQuery } from '@pinia/colada'
import { normalize } from '@/lib/common'

const { isShownCreateUserDrawer = false } = defineProps<{
  isShownCreateUserDrawer: boolean
}>()

const emit = defineEmits(['close-drawer'])

const { t } = useI18n()
const {
  state,
  asyncStatus,
  pageNum,
  pageSize,
  textFilter,
  debouncedTextFilter,
  organizationFilter,
  roleFilter,
  statusFilter,
  sortBy,
  sortDescending,
} = useUsers()
const loginStore = useLoginStore()
const { state: organizationFilterState, asyncStatus: organizationFilterAsyncStatus } =
  useOrganizationFilter()
const { state: roleFilterState, asyncStatus: roleFilterAsyncStatus } = useQuery({
  ...userRolesQuery,
  enabled: () => !!loginStore.jwtToken,
})

const currentUser = ref<User | undefined>()
const isShownCreateOrEditUserDrawer = ref(false)
const isShownDeleteUserModal = ref(false)
const isShownResetPasswordModal = ref(false)
const isShownPasswordChangedModal = ref(false)
const isShownImpersonateUserModal = ref(false)
const isShownSuspendUserModal = ref(false)
const isShownReactivateUserModal = ref(false)
const isShownRestoreUserModal = ref(false)
const isShownDestroyUserModal = ref(false)
const newPassword = ref<string>('')
const isImpersonating = ref(false)

const statusFilterOptions = ref<FilterOption[]>([
  {
    id: 'enabled',
    label: t('common.enabled'),
  },
  {
    id: 'suspended',
    label: t('common.suspended'),
  },
  {
    id: 'deleted',
    label: t('common.archived'),
  },
])

const usersPage = computed(() => {
  return state.value.data?.users
})

const pagination = computed(() => {
  return state.value.data?.pagination
})

const areDefaultFiltersApplied = computed(() => {
  return (
    !debouncedTextFilter.value &&
    organizationFilter.value.length === 0 &&
    roleFilter.value.length === 0 &&
    statusFilter.value.length === 2 &&
    statusFilter.value.includes('enabled') &&
    statusFilter.value.includes('suspended') &&
    !statusFilter.value.includes('deleted')
  )
})

const isNoDataEmptyStateShown = computed(() => {
  return (
    !usersPage.value?.length && state.value.status === 'success' && areDefaultFiltersApplied.value
  )
})

const isNoMatchEmptyStateShown = computed(() => {
  return (
    !usersPage.value?.length && state.value.status === 'success' && !areDefaultFiltersApplied.value
  )
})

const noEmptyStateShown = computed(() => {
  return !isNoDataEmptyStateShown.value && !isNoMatchEmptyStateShown.value
})

const organizationFilterOptions = computed(() => {
  if (!organizationFilterState.value.data || !organizationFilterState.value.data.organizations) {
    return []
  } else {
    return organizationFilterState.value.data.organizations.map((org) => ({
      id: org.id,
      label: org.name,
    }))
  }
})

const roleFilterOptions = computed(() => {
  if (!roleFilterState.value.data) {
    return []
  }

  return roleFilterState.value.data?.map((role) => ({
    id: role.id,
    label: t(`user_roles.${normalize(role.name)}`),
    description: t(`user_roles.${normalize(role.name)}_description`),
  }))
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

function resetFilters() {
  textFilter.value = ''
  organizationFilter.value = []
  roleFilter.value = []
  statusFilter.value = ['enabled', 'suspended']
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

function showSuspendUserModal(user: User) {
  currentUser.value = user
  isShownSuspendUserModal.value = true
}

function showReactivateUserModal(user: User) {
  currentUser.value = user
  isShownReactivateUserModal.value = true
}

function showRestoreUserModal(user: User) {
  currentUser.value = user
  isShownRestoreUserModal.value = true
}

function showDestroyUserModal(user: User) {
  currentUser.value = user
  isShownDestroyUserModal.value = true
}

function showImpersonateUserModal(user: User) {
  currentUser.value = user
  isShownImpersonateUserModal.value = true
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
  let items: NeDropdownItem[] = []

  // Hide impersonate option for yourself
  if (
    canImpersonateUsers() &&
    user.logto_id !== loginStore.userInfo?.logto_id &&
    !user.suspended_at &&
    !user.deleted_at
  ) {
    items = [
      ...items,
      {
        id: 'impersonate',
        label: t('users.impersonate_user'),
        icon: faUserSecret,
        action: () => showImpersonateUserModal(user),
        disabled:
          asyncStatus.value === 'loading' || isImpersonating.value || !user.can_be_impersonated,
      },
    ]
  }

  if (canManageUsers()) {
    if (user.deleted_at) {
      items = [
        ...items,
        {
          id: 'restoreUser',
          label: t('common.restore'),
          icon: faRotateLeft,
          action: () => showRestoreUserModal(user),
          disabled: asyncStatus.value === 'loading',
        },
      ]
    } else if (user.suspended_at) {
      items = [
        ...items,
        {
          id: 'reactivateUser',
          label: t('users.reactivate'),
          icon: faCirclePlay,
          action: () => showReactivateUserModal(user),
          disabled: asyncStatus.value === 'loading',
        },
        {
          id: 'deleteAccount',
          label: t('common.archive'),
          icon: faBoxArchive,
          danger: true,
          action: () => showDeleteUserModal(user),
          disabled: asyncStatus.value === 'loading',
        },
      ]
    } else {
      items = [
        ...items,
        {
          id: 'suspendUser',
          label: t('common.suspend'),
          icon: faCirclePause,
          action: () => showSuspendUserModal(user),
          disabled: asyncStatus.value === 'loading',
        },
        {
          id: 'resetPassword',
          label: t('users.reset_password'),
          icon: faKey,
          action: () => showResetPasswordModal(user),
          disabled: asyncStatus.value === 'loading',
        },
        {
          id: 'deleteAccount',
          label: t('common.archive'),
          icon: faBoxArchive,
          danger: true,
          action: () => showDeleteUserModal(user),
          disabled: asyncStatus.value === 'loading',
        },
      ]
    }
  }

  if (canDestroyUsers()) {
    items = [
      ...items,
      {
        id: 'destroyUser',
        label: t('common.destroy'),
        icon: faBomb,
        danger: true,
        action: () => showDestroyUserModal(user),
        disabled: asyncStatus.value === 'loading',
      },
    ]
  }
  return items
}

const onSort = (payload: SortEvent) => {
  sortBy.value = payload.key as keyof User
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
      v-if="state.status === 'error'"
      kind="error"
      :title="$t('users.cannot_retrieve_users')"
      :description="state.error.message"
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
          <!-- organization filter -->
          <NeDropdownFilter
            v-model="organizationFilter"
            kind="checkbox"
            :label="t('organizations.organization')"
            :options="organizationFilterOptions"
            :disabled="
              organizationFilterAsyncStatus === 'loading' ||
              organizationFilterState.status === 'error'
            "
            show-options-filter
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
          />
          <!-- role filter -->
          <NeDropdownFilter
            v-model="roleFilter"
            kind="checkbox"
            :label="t('users.role')"
            :options="roleFilterOptions"
            :disabled="roleFilterAsyncStatus === 'loading' || roleFilterState.status === 'error'"
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
          />
          <!-- status filter -->
          <NeDropdownFilter
            v-model="statusFilter"
            kind="checkbox"
            :label="t('common.status')"
            :options="statusFilterOptions"
            :show-clear-filter="false"
            :clear-filter-label="t('ne_dropdown_filter.reset_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
          />
          <!-- sort dropdown -->
          <NeSortDropdown
            v-model:sort-key="sortBy"
            v-model:sort-descending="sortDescending"
            :label="t('sort.sort')"
            :options="[
              { id: 'name', label: t('users.name') },
              { id: 'email', label: t('users.email') },
              { id: 'organization', label: t('users.organization') },
              { id: 'status', label: t('common.status') },
            ]"
            :open-menu-aria-label="t('ne_dropdown.open_menu')"
            :sort-by-label="t('sort.sort_by')"
            :sort-direction-label="t('sort.direction')"
            :ascending-label="t('sort.ascending')"
            :descending-label="t('sort.descending')"
          />
          <NeButton kind="tertiary" @click="resetFilters">
            {{ t('common.reset_filters') }}
          </NeButton>
        </div>
        <!-- update indicator -->
        <div
          v-if="asyncStatus === 'loading' && state.status !== 'pending'"
          class="flex items-center gap-2"
        >
          <NeSpinner color="white" />
          <div class="text-gray-500 dark:text-gray-400">
            {{ $t('common.updating') }}
          </div>
        </div>
      </div>
    </div>
    <!-- empty state -->
    <NeEmptyState
      v-if="isNoDataEmptyStateShown"
      :title="$t('users.no_user')"
      :icon="faUserGroup"
      class="bg-white dark:bg-gray-950"
    />
    <!-- no user matching filter -->
    <NeEmptyState
      v-else-if="isNoMatchEmptyStateShown"
      :title="$t('users.no_user_found')"
      :description="$t('common.try_changing_search_filters')"
      :icon="faCircleInfo"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton kind="tertiary" @click="resetFilters"> {{ $t('common.clear_filters') }}</NeButton>
    </NeEmptyState>
    <NeTable
      v-if="noEmptyStateShown"
      :sort-key="sortBy"
      :sort-descending="sortDescending"
      :aria-label="$t('users.title')"
      card-breakpoint="xl"
      :loading="state.status === 'pending'"
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
        <NeTableHeadCell sortable column-key="status" @sort="onSort">{{
          $t('common.status')
        }}</NeTableHeadCell>
        <NeTableHeadCell>
          <!-- no header for actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="(item, index) in usersPage" :key="index">
          <NeTableCell :data-label="$t('users.name')" :class="{ 'opacity-50': item.deleted_at }">
            {{ item.name }}
          </NeTableCell>
          <NeTableCell
            :data-label="$t('users.email')"
            class="break-all xl:break-normal"
            :class="{ 'opacity-50': item.deleted_at }"
          >
            {{ item.email }}
          </NeTableCell>
          <NeTableCell :data-label="$t('users.organization')">
            <div class="flex items-center gap-2" :class="{ 'opacity-50': item.deleted_at }">
              <NeTooltip
                v-if="item.organization.type"
                placement="top"
                trigger-event="mouseenter focus"
                class="shrink-0"
              >
                <template #trigger>
                  <OrganizationIcon :org-type="item.organization.type" size="sm" />
                </template>
                <template #content>
                  {{ t(`organizations.${item.organization.type}`) }}
                </template>
              </NeTooltip>
              {{ item.organization.name || '-' }}
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('users.roles')">
            <span
              v-if="!item.roles || item.roles.length === 0"
              :class="{ 'opacity-50': item.deleted_at }"
              >-</span
            >
            <div v-else class="flex flex-wrap gap-1" :class="{ 'opacity-50': item.deleted_at }">
              <UserRoleBadge
                v-for="role in item.roles?.sort(sortByProperty('name'))"
                :key="role.id"
                :role="role.name"
              />
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('common.status')">
            <div class="flex items-center gap-2">
              <template v-if="item.deleted_at">
                <FontAwesomeIcon
                  :icon="faBoxArchive"
                  class="size-4 text-gray-700 dark:text-gray-400"
                  aria-hidden="true"
                />
                <span>
                  {{ t('common.archived') }}
                </span>
              </template>
              <template v-else-if="item.suspended_at">
                <FontAwesomeIcon
                  :icon="faCirclePause"
                  class="size-4 text-gray-700 dark:text-gray-400"
                  aria-hidden="true"
                />
                <span>
                  {{ t('users.suspended') }}
                </span>
              </template>
              <template v-else>
                <FontAwesomeIcon
                  :icon="faCircleCheck"
                  class="size-4 text-green-600 dark:text-green-400"
                  aria-hidden="true"
                />
                <span>
                  {{ t('common.enabled') }}
                </span>
              </template>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('common.actions')">
            <div v-if="canManageUsers()" class="-ml-2.5 flex gap-2 xl:ml-0 xl:justify-end">
              <NeButton
                v-if="!item.deleted_at"
                kind="tertiary"
                @click="showEditUserDrawer(item)"
                :disabled="asyncStatus === 'loading'"
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
          :current-page="pageNum"
          :total-rows="pagination?.total_count || 0"
          :page-size="pageSize"
          :page-sizes="[5, 10, 25, 50, 100]"
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
              savePageSizeToStorage(USERS_TABLE_ID, size)
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
    <!-- suspend user modal -->
    <SuspendUserModal
      :visible="isShownSuspendUserModal"
      :user="currentUser"
      @close="isShownSuspendUserModal = false"
    />
    <!-- reactivate user modal -->
    <ReactivateUserModal
      :visible="isShownReactivateUserModal"
      :user="currentUser"
      @close="isShownReactivateUserModal = false"
    />
    <!-- restore user modal -->
    <RestoreUserModal
      :visible="isShownRestoreUserModal"
      :user="currentUser"
      @close="isShownRestoreUserModal = false"
    />
    <!-- impersonate user modal -->
    <ImpersonateUserModal
      :visible="isShownImpersonateUserModal"
      :user="currentUser"
      @close="isShownImpersonateUserModal = false"
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
    <!-- destroy user modal -->
    <DestroyUserModal
      :visible="isShownDestroyUserModal"
      :user="currentUser"
      @close="isShownDestroyUserModal = false"
    />
  </div>
</template>
