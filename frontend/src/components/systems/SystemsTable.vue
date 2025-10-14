<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  faCircleInfo,
  faCirclePlus,
  faTrash,
  faServer,
  faEye,
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
} from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import { canManageSystems } from '@/lib/permissions'
import { useLoginStore } from '@/stores/login'
import { normalize } from '@/lib/common'
import { useSystems } from '@/queries/systems'
import { SYSTEMS_TABLE_ID, type System } from '@/lib/systems'
import router from '@/router'
import CreateOrEditSystemDrawer from './CreateOrEditSystemDrawer.vue'
import DeleteSystemModal from './DeleteSystemModal.vue'

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

const loginStore = useLoginStore() //// remove?

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

////
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

const goToSystemDetails = (system: System) => {
  router.push({ name: 'system', params: { id: system.id } })
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

    state.status {{ state.status }} ////

    <!-- empty state -->
    <NeEmptyState
      v-if="state.status === 'success' && !systemsPage?.length && !debouncedTextFilter"
      :title="$t('systems.no_systems')"
      :icon="faServer"
      class="bg-white dark:bg-gray-950"
    >
      <!-- create system -->
      <NeButton
        v-if="canManageSystems()"
        kind="secondary"
        size="lg"
        class="shrink-0"
        @click="showCreateSystemDrawer()"
      >
        <template #prefix>
          <FontAwesomeIcon :icon="faCirclePlus" aria-hidden="true" />
        </template>
        {{ $t('systems.create_system') }}
      </NeButton>
    </NeEmptyState>
    <!-- no system matching filter -->
    <NeEmptyState
      v-else-if="state.status === 'success' && !systemsPage?.length && debouncedTextFilter"
      :title="$t('systems.no_systems_found')"
      :description="$t('common.try_changing_search_filters')"
      :icon="faCircleInfo"
      class="bg-white dark:bg-gray-950"
    >
      <NeButton kind="tertiary" @click="clearFilters">
        {{ $t('common.clear_filters') }}
      </NeButton>
    </NeEmptyState>
    <!-- //// check breakpoint, skeleton-columns -->
    <NeTable
      v-else
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
        <NeTableRow v-for="(item, index) in systemsPage" :key="index">
          <NeTableCell :data-label="$t('systems.name')">
            {{ item.name || '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('systems.version')" class="break-all xl:break-normal">
            {{ item.version || '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('common.fqdn_ip_address')" class="break-all xl:break-normal">
            <div v-if="item.fqdn">{{ item.fqdn }}</div>
            <div v-if="item.ipv4_address">{{ item.ipv4_address }}</div>
            <div v-if="item.ipv6_address">{{ item.ipv6_address }}</div>
            <div v-if="!item.fqdn && !item.ipv4_address && !item.ipv6_address">-</div>
          </NeTableCell>
          <NeTableCell :data-label="$t('systems.organization')">
            {{ item.organization_name || '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('systems.created_by')">
            {{ item.created_by?.user_name || '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('systems.status')">
            {{ item.status || '-' }} ////
          </NeTableCell>
          <NeTableCell :data-label="$t('common.actions')">
            <div class="-ml-2.5 flex gap-2 xl:ml-0 xl:justify-end">
              <NeButton
                kind="tertiary"
                @click="goToSystemDetails(item)"
                :disabled="asyncStatus === 'loading'"
              >
                <template #prefix>
                  <FontAwesomeIcon :icon="faEye" class="h-4 w-4" aria-hidden="true" />
                </template>
                {{ $t('common.view_details') }}
              </NeButton>
              <!-- kebab menu -->
              <NeDropdown
                v-if="canManageSystems()"
                :items="getKebabMenuItems(item)"
                :align-to-right="true"
              />
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
              savePageSizeToStorage(SYSTEMS_TABLE_ID, size)
            }
          "
        />
      </template>
    </NeTable>
    <!-- side drawer -->
    <CreateOrEditSystemDrawer
      :is-shown="isShownCreateOrEditSystemDrawer"
      :current-system="currentSystem"
      @close="onCloseDrawer"
    />
    <!-- delete system modal -->
    <DeleteSystemModal
      :visible="isShownDeleteSystemModal"
      :system="currentSystem"
      @close="isShownDeleteSystemModal = false"
    />
  </div>
</template>
