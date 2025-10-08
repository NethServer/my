<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { USERS_TABLE_ID, type User } from '@/lib/users'
import {
  faCircleInfo,
  faCirclePlus,
  faUserGroup,
  faPenToSquare,
  faTrash,
  faKey,
  faUserSecret,
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
  NeBadge,
  sortByProperty,
} from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import CreateOrEditUserDrawer from './CreateOrEditUserDrawer.vue'
import { useI18n } from 'vue-i18n'
import DeleteUserModal from './DeleteUserModal.vue'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import ResetPasswordModal from './ResetPasswordModal.vue'
import PasswordChangedModal from './PasswordChangedModal.vue'
import { useUsers } from '@/queries/users'
import { canManageUsers, canImpersonateUsers } from '@/lib/permissions'
import { useLoginStore } from '@/stores/login'
import ImpersonateUserModal from './ImpersonateUserModal.vue'
import { normalize } from '@/lib/common'
import { useSystems } from '@/queries/systems'
import type { System } from '@/lib/systems'

//// review, search for "users"

const { isShownCreateSystemDrawer = false } = defineProps<{
  isShownCreateSystemDrawer: boolean
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
  sortBy,
  sortDescending,
} = useSystems()

const loginStore = useLoginStore()

const currentSystem = ref<System | undefined>()
const isShownCreateOrEditSystemDrawer = ref(false)
const isShownDeleteSystemModal = ref(false)

const systemsPage = computed(() => {
  return state.value.data?.systems
})

const pagination = computed(() => {
  return state.value.data?.pagination
})

watch(
  () => isShownCreateSystemDrawer,
  () => {
    if (isShownCreateSystemDrawer) {
      showCreateSystemDrawer()
    }
  },
  { immediate: true },
)

function clearFilters() {
  textFilter.value = ''
}

function showCreateSystemDrawer() {
  currentSystem.value = undefined
  isShownCreateOrEditSystemDrawer.value = true
}

function showEditSystemDrawer(system: System) {
  currentSystem.value = system
  isShownCreateOrEditSystemDrawer.value = true
}

function showDeleteSystemModal(system: System) {
  currentSystem.value = system
  isShownDeleteSystemModal.value = true
}

function onCloseDrawer() {
  isShownCreateOrEditSystemDrawer.value = false
  emit('close-drawer')
}

function getKebabMenuItems(system: System) {
  const items = [
    // { ////
    //   id: 'resetPassword',
    //   label: t('users.reset_password'),
    //   icon: faKey,
    //   action: () => showResetPasswordModal(system),
    //   disabled: asyncStatus.value === 'loading',
    // },
    {
      id: 'deleteAccount',
      label: t('common.delete'),
      icon: faTrash,
      danger: true,
      action: () => showDeleteSystemModal(system),
      disabled: asyncStatus.value === 'loading',
    },
  ]
  return items
}

const onSort = (payload: SortEvent) => {
  sortBy.value = payload.key as keyof System
  sortDescending.value = payload.descending
}
</script>

<template>
  <div>
    <!-- get systems error notification -->
    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="$t('systems.cannot_retrieve_systems')"
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
            :placeholder="$t('systems.filter_systems')"
            class="max-w-48 sm:max-w-sm"
          />
          <!-- //// TODO: other filters -->
          <NeSortDropdown
            v-model:sort-key="sortBy"
            v-model:sort-descending="sortDescending"
            :label="t('sort.sort')"
            :options="[
              { id: 'name', label: t('systems.name') },
              { id: 'version', label: t('systems.version') },
              { id: 'organization_name', label: t('systems.organization') },
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
    <!-- //// check breakpoint, skeleton-columns -->
    <NeTable
      :sort-key="sortBy"
      :sort-descending="sortDescending"
      :aria-label="$t('systems.title')"
      card-breakpoint="xl"
      :loading="state.status === 'pending'"
      :skeleton-columns="5"
      :skeleton-rows="7"
    >
      <NeTableHead>
        <NeTableHeadCell sortable column-key="name" @sort="onSort">{{
          $t('systems.name')
        }}</NeTableHeadCell>
        <NeTableHeadCell sortable column-key="version" @sort="onSort">{{
          $t('systems.version')
        }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('systems.fqdn_ip_address') }}</NeTableHeadCell>
        <NeTableHeadCell sortable column-key="organization_name" @sort="onSort">{{
          $t('systems.organization')
        }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('systems.created_by') }}</NeTableHeadCell>
        <NeTableHeadCell sortable column-key="status" @sort="onSort">{{
          $t('systems.status')
        }}</NeTableHeadCell>
        <NeTableHeadCell>
          <!-- no header for actions -->
        </NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <!-- empty state -->
        <NeTableRow v-if="!systemsPage?.length && !debouncedTextFilter">
          <NeTableCell colspan="5">
            <NeEmptyState
              :title="$t('users.no_user')"
              :icon="faUserGroup"
              class="bg-white dark:bg-gray-950"
            >
              <!-- create user -->
              <NeButton
                v-if="canManageUsers()"
                kind="secondary"
                size="lg"
                class="shrink-0"
                @click="showCreateSystemDrawer()"
              >
                <template #prefix>
                  <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
                </template>
                {{ $t('users.create_user') }}
              </NeButton>
            </NeEmptyState>
          </NeTableCell>
        </NeTableRow>
        <!-- no user matching filter -->
        <NeTableRow v-else-if="!systemsPage?.length && debouncedTextFilter">
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
        <NeTableRow v-for="(item, index) in systemsPage" v-else :key="index">
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
            <span v-if="!item.roles || item.roles.length === 0">-</span>
            <div v-else class="flex flex-wrap gap-1">
              <NeBadge
                v-for="role in item.roles?.sort(sortByProperty('name'))"
                :key="role.id"
                :text="t(`user_roles.${normalize(role.name)}`)"
                kind="custom"
                customColorClasses="bg-indigo-100 text-indigo-800 dark:bg-indigo-700 dark:text-indigo-100"
                class="inline-block"
              ></NeBadge>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('common.actions')">
            <div v-if="canManageUsers()" class="-ml-2.5 flex gap-2 xl:ml-0 xl:justify-end">
              <NeButton
                kind="tertiary"
                @click="showEditSystemDrawer(item)"
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
      :is-shown="isShownCreateOrEditSystemDrawer"
      :current-user="currentSystem"
      @close="onCloseDrawer"
    />
    <!-- delete user modal -->
    <DeleteUserModal
      :visible="isShownDeleteSystemModal"
      :user="currentSystem"
      @close="isShownDeleteSystemModal = false"
    />
    <!-- impersonate user modal -->
    <ImpersonateUserModal
      :visible="isShownImpersonateUserModal"
      :user="currentSystem"
      @close="isShownImpersonateUserModal = false"
    />
    <!-- reset password modal -->
    <ResetPasswordModal
      :visible="isShownResetPasswordModal"
      :user="currentSystem"
      @close="isShownResetPasswordModal = false"
      @password-changed="onPasswordChanged"
    />
    <!-- password changed modal -->
    <PasswordChangedModal
      :visible="isShownPasswordChangedModal"
      :user="currentSystem"
      :new-password="newPassword"
      @close="onClosePasswordChangedModal"
    />
  </div>
</template>
