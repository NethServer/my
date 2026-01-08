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
  faPenToSquare,
  faCircleCheck,
  faCircleQuestion,
  faTriangleExclamation,
  faCircleXmark,
  faFilePdf,
  faFileCsv,
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
  NeButton,
  NeEmptyState,
  NeInlineNotification,
  NeTextInput,
  NeSpinner,
  NeDropdown,
  type SortEvent,
  NeSortDropdown,
  type FilterOption,
  NeDropdownFilter,
  NeTooltip,
  type NeDropdownItem,
} from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import { canManageSystems } from '@/lib/permissions'
import { useSystems } from '@/queries/systems/systems'
import {
  exportSystem,
  getExport,
  getProductLogo,
  getProductName,
  SYSTEMS_TABLE_ID,
  type System,
} from '@/lib/systems/systems'
import router from '@/router'
import CreateOrEditSystemDrawer from './CreateOrEditSystemDrawer.vue'
import DeleteSystemModal from './DeleteSystemModal.vue'
import { useProductFilter } from '@/queries/systems/productFilter'
import { useCreatedByFilter } from '@/queries/systems/createdByFilter'
import { useVersionFilter } from '@/queries/systems/versionFilter'
import UserAvatar from '../UserAvatar.vue'
import { buildVersionFilterOptions } from '@/lib/systems/versionFilter'
import OrganizationIcon from '../OrganizationIcon.vue'
import { downloadFile } from '@/lib/common'
import RegenerateSecretModal from './RegenerateSecretModal.vue'
import SecretRegeneratedModal from './SecretRegeneratedModal.vue'
import ClickToCopy from '../ClickToCopy.vue'

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
  productFilter,
  createdByFilter,
  versionFilter,
  statusFilter,
  sortBy,
  sortDescending,
} = useSystems()
const { state: productFilterState, asyncStatus: productFilterAsyncStatus } = useProductFilter()
const { state: createdByFilterState, asyncStatus: createdByFilterAsyncStatus } =
  useCreatedByFilter()
const { state: versionFilterState, asyncStatus: versionFilterAsyncStatus } = useVersionFilter()

const currentSystem = ref<System | undefined>()
const isShownCreateOrEditSystemDrawer = ref(false)
const isShownDeleteSystemModal = ref(false)
const isShownRegenerateSecretModal = ref(false)
const isShownSecretRegeneratedModal = ref(false)
const newSecret = ref<string>('')

const statusFilterOptions = ref<FilterOption[]>([
  {
    id: 'online',
    label: t('systems.status_online'),
  },
  {
    id: 'offline',
    label: t('systems.status_offline'),
  },
  {
    id: 'unknown',
    label: t('systems.status_unknown'),
  },
  { id: 'deleted', label: t('systems.status_deleted') },
])

const systemsPage = computed(() => {
  return state.value.data?.systems
})

const pagination = computed(() => {
  return state.value.data?.pagination
})

const productFilterOptions = computed(() => {
  if (!productFilterState.value.data || !productFilterState.value.data.products) {
    return []
  } else {
    return productFilterState.value.data.products.map((productId) => ({
      id: productId,
      label: getProductName(productId),
    }))
  }
})

const versionFilterOptions = computed(() => {
  if (!versionFilterState.value.data || !versionFilterState.value.data.versions) {
    return []
  } else {
    if (productFilter.value.length === 0) {
      // no product selected, show all versions
      return buildVersionFilterOptions(versionFilterState.value.data.versions)
    }

    // filter versions based on selected products
    const productVersions = versionFilterState.value.data.versions.filter((el) =>
      productFilter.value.includes(el.product),
    )
    return buildVersionFilterOptions(productVersions)
  }
})

const createdByFilterOptions = computed(() => {
  if (!createdByFilterState.value.data || !createdByFilterState.value.data.created_by) {
    return []
  } else {
    return createdByFilterState.value.data.created_by.map((user) => ({
      id: user.user_id,
      label: user.name,
    }))
  }
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

watch(
  () => productFilter.value,
  () => {
    // reset version filter when product filter changes
    versionFilter.value = []
  },
)

function resetFilters() {
  textFilter.value = ''
  productFilter.value = []
  versionFilter.value = []
  createdByFilter.value = []
  statusFilter.value = ['online', 'offline', 'unknown']
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

function showRegenerateSecretModal(system: System) {
  currentSystem.value = system
  isShownRegenerateSecretModal.value = true
}

function onCloseDrawer() {
  isShownCreateOrEditSystemDrawer.value = false
  emit('close-drawer')
}

function getKebabMenuItems(system: System) {
  let items: NeDropdownItem[] = []

  if (canManageSystems()) {
    items.push({
      id: 'editSystem',
      label: t('common.edit'),
      icon: faPenToSquare,
      action: () => showEditSystemDrawer(system),
      disabled: asyncStatus.value === 'loading',
    })
  }

  items = [
    ...items,
    {
      id: 'exportToPdf',
      label: t('systems.export_to_pdf'),
      icon: faFilePdf,
      action: () => exportSystem(system, 'pdf'),
      disabled: asyncStatus.value === 'loading',
    },
    {
      id: 'exportToCsv',
      label: t('systems.export_to_csv'),
      icon: faFileCsv,
      action: () => exportSystem(system, 'csv'),
      disabled: asyncStatus.value === 'loading',
    },
  ]

  if (canManageSystems()) {
    items = [
      ...items,
      {
        id: 'regenerateSecret',
        label: t('systems.regenerate_secret'),
        icon: faKey,
        action: () => showRegenerateSecretModal(system),
        disabled: asyncStatus.value === 'loading',
      },
      {
        id: 'deleteSystem',
        label: t('common.delete'),
        icon: faTrash,
        danger: true,
        action: () => showDeleteSystemModal(system),
        disabled: asyncStatus.value === 'loading',
      },
      //// add restore deleted system action
    ]
  }
  return items
}

const onSort = (payload: SortEvent) => {
  sortBy.value = payload.key as keyof System
  sortDescending.value = payload.descending
}

const goToSystemDetails = (system: System) => {
  router.push({ name: 'system_detail', params: { systemId: system.id } })
}

function onSecretRegenerated(secret: string) {
  newSecret.value = secret
  isShownSecretRegeneratedModal.value = true
}

function onCloseSecretRegeneratedModal() {
  isShownSecretRegeneratedModal.value = false
  newSecret.value = ''
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
      <div class="flex w-full items-end justify-between gap-4">
        <!-- filters -->
        <div class="flex flex-wrap items-center gap-4">
          <!-- text filter -->
          <NeTextInput
            v-model.trim="textFilter"
            is-search
            :placeholder="$t('systems.filter_systems')"
            class="max-w-48 sm:max-w-sm"
          />
          <NeDropdownFilter
            v-model="productFilter"
            kind="checkbox"
            :disabled="
              productFilterAsyncStatus === 'loading' || productFilterState.status === 'error'
            "
            :label="t('systems.product')"
            :options="productFilterOptions"
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
          />
          <NeDropdownFilter
            v-model="versionFilter"
            kind="checkbox"
            :disabled="
              versionFilterAsyncStatus === 'loading' || versionFilterState.status === 'error'
            "
            :label="t('systems.version')"
            :options="versionFilterOptions"
            show-options-filter
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
          />
          <NeDropdownFilter
            v-model="createdByFilter"
            kind="checkbox"
            :disabled="
              createdByFilterAsyncStatus === 'loading' || createdByFilterState.status === 'error'
            "
            :label="t('systems.created_by')"
            :options="createdByFilterOptions"
            show-options-filter
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
          />
          <NeDropdownFilter
            v-model="statusFilter"
            kind="checkbox"
            :label="t('common.status')"
            :options="statusFilterOptions"
            :show-clear-filter="false"
            :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
          />
          <!-- sort dropdown for small screens -->
          <NeSortDropdown
            v-model:sort-key="sortBy"
            v-model:sort-descending="sortDescending"
            :label="t('sort.sort')"
            :options="[
              { id: 'name', label: t('systems.name') },
              { id: 'version', label: t('systems.version') },
              { id: 'fqdn', label: t('systems.fqdn') },
              { id: 'organization_name', label: t('systems.organization') },
              { id: 'creator_name', label: t('systems.created_by') },
              { id: 'status', label: t('systems.status') },
            ]"
            :open-menu-aria-label="t('ne_dropdown.open_menu')"
            :sort-by-label="t('sort.sort_by')"
            :sort-direction-label="t('sort.direction')"
            :ascending-label="t('sort.ascending')"
            :descending-label="t('sort.descending')"
            class="2xl:hidden"
          />
          <NeButton kind="tertiary" @click="resetFilters">
            {{ t('systems.reset_filters') }}
          </NeButton>
        </div>
        <!-- update indicator -->
        <div
          v-if="asyncStatus === 'loading' && state.status !== 'pending'"
          class="relative -top-2 flex items-center gap-2"
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
      v-if="state.status === 'success' && !systemsPage?.length && !debouncedTextFilter"
      :title="$t('systems.no_systems')"
      :icon="faServer"
      class="bg-white dark:bg-gray-950"
    >
      <!-- create system -->
      <NeButton
        v-if="canManageSystems()"
        kind="primary"
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
      <NeButton kind="tertiary" @click="resetFilters">
        {{ $t('systems.reset_filters') }}
      </NeButton>
    </NeEmptyState>
    <NeTable
      v-else
      :sort-key="sortBy"
      :sort-descending="sortDescending"
      :aria-label="$t('systems.title')"
      card-breakpoint="2xl"
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
        <NeTableHeadCell sortable column-key="fqdn" @sort="onSort">{{
          $t('systems.fqdn_ip_address')
        }}</NeTableHeadCell>
        <NeTableHeadCell sortable column-key="organization_name" @sort="onSort">{{
          $t('systems.organization')
        }}</NeTableHeadCell>
        <NeTableHeadCell sortable column-key="creator_name" @sort="onSort">{{
          $t('systems.created_by')
        }}</NeTableHeadCell>
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
            <div :class="{ 'opacity-50': item.status === 'deleted' }">
              <router-link :to="{ name: 'system_detail', params: { systemId: item.id } }">
                <div class="flex items-center gap-2">
                  <img
                    v-if="item.type"
                    :src="getProductLogo(item.type)"
                    :alt="getProductName(item.type)"
                    aria-hidden="true"
                    class="size-8"
                  />
                  <span class="cursor-pointer font-medium hover:underline">
                    {{ item.name || '-' }}
                  </span>
                </div>
              </router-link>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('systems.version')" class="break-all 2xl:break-normal">
            <div :class="{ 'opacity-50': item.status === 'deleted' }">
              {{ item.version || '-' }}
            </div>
          </NeTableCell>
          <NeTableCell
            :data-label="$t('systems.fqdn_ip_address')"
            class="break-all 2xl:break-normal"
          >
            <div
              class="flex flex-col items-start space-y-0.5"
              :class="{ 'opacity-50': item.status === 'deleted' }"
            >
              <ClickToCopy v-if="item.fqdn" :text="item.fqdn" tooltip-placement="top" />
              <ClickToCopy
                v-if="item.ipv4_address"
                :text="item.ipv4_address"
                tooltip-placement="bottom"
              />
              <div v-if="item.ipv6_address">
                {{ item.ipv6_address }}
              </div>
              <div v-if="!item.fqdn && !item.ipv4_address && !item.ipv6_address">-</div>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('systems.organization')">
            <div :class="{ 'opacity-50': item.status === 'deleted' }">
              <div class="flex items-center gap-2">
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
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('systems.created_by')">
            <div :class="{ 'opacity-50': item.status === 'deleted' }">
              <template v-if="item.created_by">
                <div class="flex items-center gap-2">
                  <UserAvatar
                    size="sm"
                    :is-owner="item.created_by.username === 'owner'"
                    :name="item.created_by.name"
                  />
                  <div class="space-y-0.5">
                    <div>{{ item.created_by.name || '-' }}</div>
                    <div
                      v-if="item.created_by.organization_name"
                      class="text-gray-500 dark:text-gray-400"
                    >
                      {{ item.created_by.organization_name }}
                    </div>
                  </div>
                </div>
              </template>
              <template v-else>-</template>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('systems.status')">
            <div class="flex items-center gap-2">
              <FontAwesomeIcon
                v-if="item.status === 'online'"
                :icon="faCircleCheck"
                class="size-4 text-green-600 dark:text-green-400"
                aria-hidden="true"
              />
              <FontAwesomeIcon
                v-else-if="item.status === 'offline'"
                :icon="faTriangleExclamation"
                class="size-4 text-amber-700 dark:text-amber-500"
                aria-hidden="true"
              />
              <FontAwesomeIcon
                v-else-if="item.status === 'deleted'"
                :icon="faCircleXmark"
                class="size-4 text-rose-700 dark:text-rose-500"
                aria-hidden="true"
              />
              <FontAwesomeIcon
                v-else
                :icon="faCircleQuestion"
                class="size-4 text-gray-700 dark:text-gray-400"
                aria-hidden="true"
              />
              <span v-if="item.status">
                {{ t(`systems.status_${item.status}`) }}
              </span>
              <span v-else>-</span>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('common.actions')">
            <div
              v-if="item.status !== 'deleted'"
              class="-ml-2.5 flex gap-2 2xl:ml-0 2xl:justify-end"
            >
              <NeButton
                kind="tertiary"
                @click="goToSystemDetails(item)"
                :disabled="asyncStatus === 'loading' || item.status === 'deleted'"
              >
                <template #prefix>
                  <FontAwesomeIcon :icon="faEye" class="h-4 w-4" aria-hidden="true" />
                </template>
                {{ $t('common.view_details') }}
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
    <!-- regenerate secret modal -->
    <RegenerateSecretModal
      :visible="isShownRegenerateSecretModal"
      :system="currentSystem"
      @close="isShownRegenerateSecretModal = false"
      @secret-regenerated="onSecretRegenerated"
    />
    <!-- secret regenerated modal -->
    <SecretRegeneratedModal
      :visible="isShownSecretRegeneratedModal"
      :system="currentSystem"
      :new-secret="newSecret"
      @close="onCloseSecretRegeneratedModal"
    />
  </div>
</template>
