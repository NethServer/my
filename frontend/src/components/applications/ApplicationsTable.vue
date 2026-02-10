<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  faBuilding,
  faCircleInfo,
  faEye,
  faPenToSquare,
  faServer,
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
  NeTooltip,
  type NeDropdownItem,
  NeDropdownFilter,
  NeSortDropdown,
} from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import { canManageApplications } from '@/lib/permissions'
import { SYSTEMS_TABLE_ID } from '@/lib/systems/systems'
import router from '@/router'
import OrganizationIcon from '../OrganizationIcon.vue'
import { useApplications } from '@/queries/applications'
import {
  getApplicationLogo,
  getDisplayName,
  type Application,
} from '@/lib/applications/applications'
import { faGridOne } from '@nethesis/nethesis-solid-svg-icons'
import AssignOrganizationDrawer from './AssignOrganizationDrawer.vue'
import SetNotesDrawer from './SetNotesDrawer.vue'
import { useTypeFilter } from '@/queries/applications/typeFilter'
import { useVersionFilter } from '@/queries/applications/versionFilter'
import { buildVersionFilterOptions } from '@/lib/applications/versionFilter'
import { useSystemFilter } from '@/queries/applications/systemFilter'
import { useOrganizationFilter } from '@/queries/applications/organizationFilter'

//// review (search "system")

const { t } = useI18n()
const {
  state,
  asyncStatus,
  pageNum,
  pageSize,
  textFilter,
  debouncedTextFilter,
  typeFilter,
  versionFilter,
  systemFilter,
  organizationFilter,
  sortBy,
  sortDescending,
} = useApplications()

const { state: typeFilterState, asyncStatus: typeFilterAsyncStatus } = useTypeFilter()
const { state: versionFilterState, asyncStatus: versionFilterAsyncStatus } = useVersionFilter()
const { state: systemFilterState, asyncStatus: systemFilterAsyncStatus } = useSystemFilter()
const { state: organizationFilterState, asyncStatus: organizationFilterAsyncStatus } =
  useOrganizationFilter()

const currentApplication = ref<Application | undefined>()
const isShownAssignOrgDrawer = ref(false)
const isShownSetNotesDrawer = ref(false)

const applicationsPage = computed(() => {
  return state.value.data?.applications || []
})

const pagination = computed(() => {
  return state.value.data?.pagination
})

const typeFilterOptions = computed(() => {
  if (!typeFilterState.value.data || !typeFilterState.value.data) {
    return []
  } else {
    return typeFilterState.value.data.map((appType) => ({
      id: appType.instance_of,
      label: appType.instance_of,
    }))
  }
})

const versionFilterOptions = computed(() => {
  if (!versionFilterState.value.data || !versionFilterState.value.data.versions) {
    return []
  } else {
    if (typeFilter.value.length === 0) {
      // no application selected, show all versions
      return buildVersionFilterOptions(versionFilterState.value.data.versions)
    }

    // filter versions based on selected applications
    const applicationVersions = versionFilterState.value.data.versions.filter((el) =>
      typeFilter.value.includes(el.application),
    )
    return buildVersionFilterOptions(applicationVersions)
  }
})

const systemFilterOptions = computed(() => {
  if (!systemFilterState.value.data || !systemFilterState.value.data) {
    return []
  } else {
    return systemFilterState.value.data.map((appSystem) => ({
      id: appSystem.id,
      label: appSystem.name,
    }))
  }
})

const organizationFilterOptions = computed(() => {
  if (!organizationFilterState.value.data) {
    return []
  } else {
    return organizationFilterState.value.data.map((org) => ({
      id: org.logto_id,
      label: org.logto_id === 'no_org' ? t('applications.no_organization') : org.name,
    }))
  }
})

const isFiltered = computed(() => {
  return (
    !!debouncedTextFilter.value ||
    !!typeFilter.value.length ||
    !!versionFilter.value.length ||
    !!systemFilter.value.length ||
    !!organizationFilter.value.length
  )
})

const isNoDataEmptyStateShown = computed(() => {
  return !applicationsPage.value?.length && state.value.status === 'success' && !isFiltered.value
})

const isNoMatchEmptyStateShown = computed(() => {
  return !applicationsPage.value?.length && state.value.status === 'success' && !!isFiltered.value
})

const noEmptyStateShown = computed(() => {
  return !isNoDataEmptyStateShown.value && !isNoMatchEmptyStateShown.value
})

watch(
  () => typeFilter.value,
  () => {
    // reset version filter when product filter changes
    versionFilter.value = []
  },
)

////
function clearFilters() {
  console.log('clearFilters, todo') ////

  textFilter.value = ''
  typeFilter.value = []
  versionFilter.value = []
  systemFilter.value = []
  organizationFilter.value = []
  //   createdByFilter.value = []
  //   statusFilter.value = ['online', 'offline', 'unknown']
}

function showAssignOrgDrawer(application: Application) {
  currentApplication.value = application
  isShownAssignOrgDrawer.value = true
}

function showSetNotesDrawer(application: Application) {
  currentApplication.value = application
  isShownSetNotesDrawer.value = true
}

function getKebabMenuItems(application: Application) {
  const items: NeDropdownItem[] = []

  if (canManageApplications()) {
    items.push({
      id: 'assignOrganization',
      label: application.organization?.logto_id
        ? t('applications.reassign_organization')
        : t('applications.assign_organization'),
      icon: faBuilding,
      action: () => showAssignOrgDrawer(application),
      disabled: asyncStatus.value === 'loading',
    })

    items.push({
      id: 'setNotes',
      label: application?.notes ? t('applications.edit_notes') : t('applications.add_notes'),
      icon: faPenToSquare,
      action: () => showSetNotesDrawer(application),
      disabled: asyncStatus.value === 'loading',
    })
  }
  return items
}

const onSort = (payload: SortEvent) => {
  sortBy.value = payload.key as keyof Application
  sortDescending.value = payload.descending
}

const goToApplicationDetails = (application: Application) => {
  router.push({ name: 'application_detail', params: { applicationId: application.id } })
}
</script>

<template>
  <div>
    <!-- get applications error notification -->
    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="$t('applications.cannot_retrieve_applications')"
      :description="state.error.message"
      class="mb-6"
    />
    <!-- empty state -->
    <NeEmptyState
      v-if="isNoDataEmptyStateShown"
      :title="$t('applications.no_applications')"
      :description="$t('applications.no_applications_description')"
      :icon="faGridOne"
      class="bg-white dark:bg-gray-950"
    />
    <template v-if="!isNoDataEmptyStateShown">
      <!-- table toolbar -->
      <div class="mb-6 flex items-center gap-4">
        <div class="flex w-full items-end justify-between gap-4">
          <!-- filters -->
          <div class="flex flex-wrap items-center gap-4">
            <!-- text filter -->
            <NeTextInput
              v-model.trim="textFilter"
              is-search
              :placeholder="$t('applications.filter_applications')"
              class="max-w-48 sm:max-w-sm"
            />
            <NeDropdownFilter
              v-model="typeFilter"
              kind="checkbox"
              :disabled="typeFilterAsyncStatus === 'loading' || typeFilterState.status === 'error'"
              :label="t('applications.type')"
              :options="typeFilterOptions"
              show-options-filter
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
              :label="t('applications.version')"
              :options="versionFilterOptions"
              show-options-filter
              :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
              :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
              :no-options-label="t('ne_dropdown_filter.no_options')"
              :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
              :clear-search-label="t('ne_dropdown_filter.clear_search')"
            />
            <NeDropdownFilter
              v-model="systemFilter"
              kind="checkbox"
              :disabled="
                systemFilterAsyncStatus === 'loading' || systemFilterState.status === 'error'
              "
              :label="t('systems.system')"
              :options="systemFilterOptions"
              show-options-filter
              :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
              :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
              :no-options-label="t('ne_dropdown_filter.no_options')"
              :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
              :clear-search-label="t('ne_dropdown_filter.clear_search')"
            />
            <NeDropdownFilter
              v-model="organizationFilter"
              kind="checkbox"
              :disabled="
                organizationFilterAsyncStatus === 'loading' ||
                organizationFilterState.status === 'error'
              "
              :label="t('organizations.organization')"
              :options="organizationFilterOptions"
              show-options-filter
              :clear-filter-label="t('ne_dropdown_filter.clear_filter')"
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
                { id: 'display_name', label: t('applications.name') },
                { id: 'instance_of', label: t('applications.type') },
                { id: 'version', label: t('applications.version') },
                { id: 'system_name', label: t('systems.system') },
                { id: 'organization_name', label: t('organizations.organization') },
              ]"
              :open-menu-aria-label="t('ne_dropdown.open_menu')"
              :sort-by-label="t('sort.sort_by')"
              :sort-direction-label="t('sort.direction')"
              :ascending-label="t('sort.ascending')"
              :descending-label="t('sort.descending')"
            />
            <NeButton kind="tertiary" @click="clearFilters">
              {{ t('common.clear_filters') }}
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
      <!-- no application matching filter -->
      <NeEmptyState
        v-if="isNoMatchEmptyStateShown"
        :title="$t('applications.no_applications_found')"
        :description="$t('common.try_changing_search_filters')"
        :icon="faCircleInfo"
        class="bg-white dark:bg-gray-950"
      >
        <NeButton kind="tertiary" @click="clearFilters">
          {{ $t('common.clear_filters') }}
        </NeButton>
      </NeEmptyState>
      <NeTable
        v-if="noEmptyStateShown"
        :sort-key="sortBy"
        :sort-descending="sortDescending"
        :aria-label="$t('applications.title')"
        card-breakpoint="2xl"
        :loading="state.status === 'pending'"
        :skeleton-columns="5"
        :skeleton-rows="7"
      >
        <NeTableHead>
          <NeTableHeadCell sortable column-key="display_name" @sort="onSort">{{
            $t('applications.name')
          }}</NeTableHeadCell>
          <NeTableHeadCell sortable column-key="instance_of" @sort="onSort">{{
            $t('applications.type')
          }}</NeTableHeadCell>
          <NeTableHeadCell sortable column-key="version" @sort="onSort">{{
            $t('applications.version')
          }}</NeTableHeadCell>
          <NeTableHeadCell sortable column-key="system_name" @sort="onSort">{{
            $t('systems.system')
          }}</NeTableHeadCell>
          <NeTableHeadCell sortable column-key="organization_name" @sort="onSort">{{
            $t('organizations.organization')
          }}</NeTableHeadCell>
          <NeTableHeadCell>
            <!-- no header for actions -->
          </NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="(item, index) in applicationsPage" :key="index">
            <NeTableCell :data-label="$t('applications.name')">
              {{ getDisplayName(item) }}
            </NeTableCell>
            <NeTableCell :data-label="$t('applications.type')">
              <!-- <router-link :to="{ name: 'application_detail', params: { applicationId: item.id } }"> //// uncomment -->
              <div class="flex items-center gap-2">
                <img
                  v-if="item.instance_of"
                  :src="getApplicationLogo(item.instance_of)"
                  :alt="item.instance_of"
                  aria-hidden="true"
                  class="size-8"
                />
                <span class="cursor-pointer font-medium hover:underline">
                  {{ item.name || '-' }}
                </span>
              </div>
              <!-- </router-link> //// -->
            </NeTableCell>
            <NeTableCell
              :data-label="$t('applications.version')"
              class="break-all 2xl:break-normal"
            >
              <div>
                {{ item.version || '-' }}
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('systems.system')">
              <div>
                <router-link :to="{ name: 'system_detail', params: { systemId: item.system.id } }">
                  <div class="flex items-center gap-2">
                    <FontAwesomeIcon :icon="faServer" class="h-4 w-4" aria-hidden="true" />
                    <span class="cursor-pointer font-medium hover:underline">
                      {{ item.system.name || '-' }}
                    </span>
                  </div>
                </router-link>
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('organizations.organization')">
              <div>
                <div class="flex items-center gap-2">
                  <NeTooltip
                    v-if="item.organization?.type"
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
                  {{ item.organization?.name || '-' }}
                </div>
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('common.actions')">
              <div class="-ml-2.5 flex gap-2 2xl:ml-0 2xl:justify-end">
                <NeButton
                  v-if="item.status !== 'deleted'"
                  kind="tertiary"
                  @click="goToApplicationDetails(item)"
                  :disabled="asyncStatus === 'loading' || item.status === 'deleted'"
                >
                  <template #prefix>
                    <FontAwesomeIcon :icon="faEye" class="h-4 w-4" aria-hidden="true" />
                  </template>
                  {{ $t('common.view') }}
                </NeButton>
                <!-- kebab menu -->
                <NeDropdown
                  v-if="canManageApplications()"
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
    </template>
    <!-- assign organization drawer -->
    <AssignOrganizationDrawer
      :is-shown="isShownAssignOrgDrawer"
      :current-application="currentApplication"
      @close="isShownAssignOrgDrawer = false"
    />
    <!-- set notes drawer -->
    <SetNotesDrawer
      :is-shown="isShownSetNotesDrawer"
      :current-application="currentApplication"
      @close="isShownSetNotesDrawer = false"
    />
  </div>
</template>
