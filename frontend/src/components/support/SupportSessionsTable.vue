<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  faHeadset,
  faClockRotateLeft,
  faXmark,
  faMagnifyingGlass,
  faWindowMaximize,
  faTerminal,
  faChevronDown,
} from '@fortawesome/free-solid-svg-icons'
import {
  NeTable,
  NeTableHead,
  NeTableHeadCell,
  NeTableBody,
  NeTableRow,
  NeTableCell,
  NePaginator,
  NeButton,
  NeDropdown,
  type NeDropdownItem,
  NeEmptyState,
  NeInlineNotification,
  NeBadge,
  NeTextInput,
  NeDropdownFilterV2,
  NeSortDropdown,
  type NeDropdownFilterV2Option,
  type SortEvent,
  formatDateTimeNoSeconds,
} from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import {
  SUPPORT_SESSIONS_SCOPE_STATUSES,
  supportSessionsTableId,
  useLiveSupportSessions,
  useSupportSessionsHistory,
  type SupportSessionsScope,
} from '@/queries/support/supportSessions'
import {
  SUPPORT_SESSIONS_KEY,
  type SupportSessionStatus,
  type SupportSessionSortBy,
  type SystemSessionGroup,
  closeSupportSession,
  getSupportSessionServices,
} from '@/lib/support/support'
import UpdatingSpinner from '@/components/common/UpdatingSpinner.vue'
import { canConnectSystems } from '@/lib/permissions'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { useQueryCache } from '@pinia/colada'
import { getProductName } from '@/lib/systems/systems'
import { useSupportWorkspacesStore, workspaceOrganizations } from '@/stores/supportWorkspaces'
import SystemLogoAndLink from '@/components/systems/SystemLogoAndLink.vue'
import OrganizationIconAndLink from '@/components/organizations/OrganizationIconAndLink.vue'
import OrganizationDropdownFilter from '@/components/organizations/OrganizationDropdownFilter.vue'
import {
  canSeeParentOfCustomers,
  getManagedBy,
  isUserCustomer,
} from '@/lib/organizations/organizations'
import ParentCompanyLink from '@/components/organizations/ParentCompanyLink.vue'
import SupportHistoryDrawer from '@/components/support/SupportHistoryDrawer.vue'
import SupportExtendModal from '@/components/support/SupportExtendModal.vue'
import { createSSHCommand } from '@/lib/support/ssh'
import { useNotificationsStore } from '@/stores/notifications'

const props = withDefaults(defineProps<{ scope?: SupportSessionsScope }>(), { scope: 'live' })

const { locale, t } = useI18n()
const queryCache = useQueryCache()
const {
  state,
  asyncStatus,
  pageNum,
  pageSize,
  textFilter,
  productFilter,
  organizationFilter,
  statusFilter,
  sortBy,
  sortDescending,
  areDefaultFiltersApplied,
  resetFilters,
} = props.scope === 'live' ? useLiveSupportSessions() : useSupportSessionsHistory()

const productFilterOptions = computed<NeDropdownFilterV2Option[]>(() =>
  ['nsec', 'ns8'].map((id) => ({ id, label: getProductName(id) })),
)

const statusFilterOptions = computed<NeDropdownFilterV2Option[]>(() =>
  SUPPORT_SESSIONS_SCOPE_STATUSES[props.scope].map((id) => ({
    id,
    label: t(`support.status_${id}`),
  })),
)

const onSort = (payload: SortEvent) => {
  sortBy.value = payload.key as SupportSessionSortBy
  sortDescending.value = payload.descending
}

const sessionGroups = computed(() => {
  return state.value.data?.support_sessions || []
})

const pagination = computed(() => {
  return state.value.data?.pagination
})

const isNoDataEmptyStateShown = computed(() => {
  return (
    !sessionGroups.value?.length &&
    state.value.status === 'success' &&
    areDefaultFiltersApplied.value
  )
})

const isNoMatchEmptyStateShown = computed(() => {
  return (
    !sessionGroups.value?.length &&
    state.value.status === 'success' &&
    !areDefaultFiltersApplied.value
  )
})

const closingSessionId = ref<string | null>(null)

function getStatusBadgeKind(status: SupportSessionStatus) {
  switch (status) {
    case 'active':
      return 'success'
    case 'pending':
      return 'warning'
    case 'expired':
      return 'secondary'
    case 'closed':
      return 'secondary'
    default:
      return 'secondary'
  }
}

// Extend: the operator picks how long, within the backend's limits
const notificationsStore = useNotificationsStore()
const extendGroup = ref<SystemSessionGroup | null>(null)

function handleExtendGroup(group: SystemSessionGroup) {
  extendGroup.value = group
}

function onExtended(group: SystemSessionGroup, expiresAt: Date) {
  queryCache.invalidateQueries({ key: [SUPPORT_SESSIONS_KEY] })
  notificationsStore.createNotification({
    kind: 'success',
    title: t('support.extended_title'),
    description: t('support.extended_description', {
      name: group.system_name || group.system_key,
      date: formatDateTimeNoSeconds(expiresAt, locale.value),
    }),
  })
}

const workspacesStore = useSupportWorkspacesStore()

// a past intervention opens in a drawer: its timeline and notes
const historyGroup = ref<SystemSessionGroup | null>(null)
const isHistoryShown = ref(false)

function handleShowHistory(group: SystemSessionGroup) {
  historyGroup.value = group
  isHistoryShown.value = true
}

// SSH: copy the command to paste in a terminal, one per node on a cluster.
// Each copy carries a fresh one-time token bound to this operator.

function sshNodes(group: SystemSessionGroup): string[] {
  return group.sessions
    .filter((s) => s.status === 'active' && s.node_id)
    .map((s) => s.node_id as string)
    .sort((a, b) => a.localeCompare(b, undefined, { numeric: true }))
}

// the labels given to the cluster nodes, read from a session's services
const nodeLabels = ref<Record<string, Record<string, string>>>({})
const nodeLabelsLoading = new Set<string>()

async function loadNodeLabels(group: SystemSessionGroup) {
  const session = group.sessions.find((s) => s.status === 'active')
  if (!session || nodeLabels.value[group.system_id] || nodeLabelsLoading.has(group.system_id))
    return
  nodeLabelsLoading.add(group.system_id)
  try {
    const labels: Record<string, string> = {}
    for (const sg of await getSupportSessionServices(session.id)) {
      for (const svc of sg.services) {
        if (svc.nodeId && svc.nodeLabel && !labels[svc.nodeId]) labels[svc.nodeId] = svc.nodeLabel
      }
    }
    nodeLabels.value = { ...nodeLabels.value, [group.system_id]: labels }
  } catch {
    // no labels: the nodes keep their number
  } finally {
    nodeLabelsLoading.delete(group.system_id)
  }
}

// the clusters on the page get their node labels for the SSH menu
watch(
  () => sessionGroups.value,
  (groups) => {
    for (const group of groups || []) {
      if (group.status === 'active' && sshNodes(group).length > 1) loadNodeLabels(group)
    }
  },
  { immediate: true },
)

async function copySSHCommand(group: SystemSessionGroup, nodeId?: string) {
  const command = createSSHCommand(group.system_id, nodeId)
  try {
    // the clipboard write starts within the click, the command follows
    if (typeof ClipboardItem !== 'undefined' && navigator.clipboard.write) {
      await navigator.clipboard.write([
        new ClipboardItem({
          'text/plain': command.then((text) => new Blob([text], { type: 'text/plain' })),
        }),
      ])
    } else {
      await navigator.clipboard.writeText(await command)
    }
    notificationsStore.createNotification({
      kind: 'success',
      title: t('support.ssh_copied'),
      description: t('support.ssh_copied_description', {
        name: group.system_name || group.system_key,
      }),
    })
  } catch {
    notificationsStore.createNotification({ kind: 'error', title: t('support.ssh_copy_error') })
  }
}

function sshDropdownItems(group: SystemSessionGroup): NeDropdownItem[] {
  const labels = nodeLabels.value[group.system_id] || {}
  return sshNodes(group).map((nodeId) => ({
    id: `ssh-${nodeId}`,
    label: labels[nodeId]
      ? t('support.ssh_copy_node_label', { node: nodeId, label: labels[nodeId] })
      : t('support.ssh_copy_node', { node: nodeId }),
    action: () => copySSHCommand(group, nodeId),
  }))
}

function handleOpenWorkspace(group: SystemSessionGroup) {
  workspacesStore.open(
    {
      systemId: group.system_id,
      systemName: group.system_name || group.system_key,
      systemType: group.system_type || null,
      ...workspaceOrganizations(group),
    },
    group.sessions,
  )
}

async function handleCloseGroup(group: SystemSessionGroup) {
  closingSessionId.value = group.system_id
  try {
    await Promise.all(group.sessions.map((s) => closeSupportSession(s.id)))
    queryCache.invalidateQueries({ key: [SUPPORT_SESSIONS_KEY] })
  } catch (error) {
    console.error('Cannot close support session:', error)
  } finally {
    closingSessionId.value = null
  }
}
</script>

<template>
  <div>
    <div class="flex flex-col gap-6">
      <!-- error notification -->
      <NeInlineNotification
        v-if="state.status === 'error'"
        kind="error"
        :title="$t('support.cannot_retrieve_sessions')"
        :description="state.error.message"
      />
      <!-- table toolbar -->
      <div class="flex w-full items-end justify-between gap-4">
        <div class="flex flex-wrap items-center gap-4">
          <NeTextInput
            v-model="textFilter"
            is-search
            :placeholder="$t('support.filter_sessions')"
            class="max-w-48 sm:max-w-sm"
            @blur="textFilter = textFilter.trim()"
          />
          <NeDropdownFilterV2
            v-model="productFilter"
            kind="checkbox"
            :label="t('systems.product')"
            :options="productFilterOptions"
            :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
            :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
          />
          <OrganizationDropdownFilter
            v-if="!isUserCustomer()"
            v-model="organizationFilter"
            :label="t('systems.organization')"
          />
          <NeDropdownFilterV2
            v-model="statusFilter"
            kind="checkbox"
            :label="t('common.status')"
            :options="statusFilterOptions"
            :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
            :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
            :no-options-label="t('ne_dropdown_filter.no_options')"
            :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
            :clear-search-label="t('ne_dropdown_filter.clear_search')"
            :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
          />
          <NeSortDropdown
            v-model:sort-key="sortBy"
            v-model:sort-descending="sortDescending"
            :label="t('sort.sort')"
            :options="[
              { id: 'name', label: t('support.system') },
              { id: 'organization_name', label: t('systems.organization') },
              { id: 'started_at', label: t('support.started_at') },
              { id: 'expires_at', label: t('support.expires_at') },
              { id: 'status', label: t('support.status') },
            ]"
            :open-menu-aria-label="t('ne_dropdown.open_menu')"
            :sort-by-label="t('sort.sort_by')"
            :sort-direction-label="t('sort.direction')"
            :ascending-label="t('sort.ascending')"
            :descending-label="t('sort.descending')"
            align-to-right
          />
          <NeButton kind="tertiary" @click="resetFilters">
            {{ t('systems.reset_filters') }}
          </NeButton>
        </div>
        <!-- update indicator -->
        <UpdatingSpinner v-if="asyncStatus === 'loading' && state.status !== 'pending'" />
      </div>
      <!-- no data empty state -->
      <NeEmptyState
        v-if="isNoDataEmptyStateShown"
        :title="scope === 'live' ? $t('support.no_sessions') : $t('support.no_sessions_history')"
        :icon="faHeadset"
        class="bg-white dark:bg-gray-950"
      />
      <!-- no session matching filters -->
      <NeEmptyState
        v-else-if="isNoMatchEmptyStateShown"
        :title="$t('support.no_sessions_found')"
        :description="$t('common.try_changing_search_filters')"
        :icon="faMagnifyingGlass"
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
        :aria-label="$t('support.title')"
        card-breakpoint="2xl"
        :loading="state.status === 'pending'"
        :skeleton-columns="6"
        :skeleton-rows="5"
      >
        <NeTableHead>
          <NeTableHeadCell sortable column-key="name" @sort="onSort">{{
            $t('support.system')
          }}</NeTableHeadCell>
          <NeTableHeadCell sortable column-key="organization_name" @sort="onSort">{{
            $t('systems.organization')
          }}</NeTableHeadCell>
          <NeTableHeadCell v-if="canSeeParentOfCustomers()">{{
            $t('organizations.parent_company')
          }}</NeTableHeadCell>
          <NeTableHeadCell sortable column-key="started_at" @sort="onSort">{{
            $t('support.started_at')
          }}</NeTableHeadCell>
          <NeTableHeadCell sortable column-key="expires_at" @sort="onSort">{{
            $t('support.expires_at')
          }}</NeTableHeadCell>
          <NeTableHeadCell sortable column-key="status" @sort="onSort">{{
            $t('support.status')
          }}</NeTableHeadCell>
          <NeTableHeadCell v-if="canConnectSystems()">
            <!-- no header for actions -->
          </NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="group in sessionGroups" :key="group.system_id">
            <NeTableCell :data-label="$t('support.system')">
              <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                <SystemLogoAndLink
                  class="min-w-36"
                  :system-id="group.system_id"
                  :system-name="group.system_name || group.system_key"
                  :system-type="group.system_type"
                />
                <NeBadge
                  v-if="group.node_count > 1"
                  class="shrink-0 whitespace-nowrap"
                  :text="$t('support.node_count', { count: group.node_count })"
                  kind="secondary"
                  size="sm"
                />
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('systems.organization')">
              <OrganizationIconAndLink
                v-if="group.organization?.name"
                :organization="group.organization"
              />
              <span v-else>-</span>
            </NeTableCell>
            <NeTableCell
              v-if="canSeeParentOfCustomers()"
              :data-label="$t('organizations.parent_company')"
            >
              <ParentCompanyLink
                v-if="group.organization && getManagedBy(group.organization)"
                :parent="getManagedBy(group.organization)"
              />
              <span v-else>-</span>
            </NeTableCell>
            <NeTableCell :data-label="$t('support.started_at')">
              {{
                group.started_at ? formatDateTimeNoSeconds(new Date(group.started_at), locale) : '-'
              }}
            </NeTableCell>
            <NeTableCell :data-label="$t('support.expires_at')">
              {{
                group.expires_at ? formatDateTimeNoSeconds(new Date(group.expires_at), locale) : '-'
              }}
            </NeTableCell>
            <NeTableCell :data-label="$t('support.status')">
              <NeBadge
                :text="$t(`support.status_${group.status}`)"
                :kind="getStatusBadgeKind(group.status)"
              />
            </NeTableCell>
            <NeTableCell v-if="canConnectSystems()" :data-label="$t('common.actions')">
              <div class="-ml-2.5 flex flex-wrap gap-2 2xl:ml-0 2xl:flex-nowrap 2xl:justify-end">
                <NeButton
                  v-if="group.status === 'active'"
                  kind="tertiary"
                  size="sm"
                  @click="handleOpenWorkspace(group)"
                >
                  <template #prefix>
                    <FontAwesomeIcon :icon="faWindowMaximize" aria-hidden="true" />
                  </template>
                  {{ $t('support.open_workspace') }}
                </NeButton>
                <NeDropdown
                  v-if="group.status === 'active' && group.system_key && sshNodes(group).length > 1"
                  :items="sshDropdownItems(group)"
                  :align-to-right="true"
                >
                  <template #button>
                    <NeButton kind="tertiary" size="sm" :title="$t('support.ssh_copy_hint')">
                      <template #prefix>
                        <FontAwesomeIcon :icon="faTerminal" aria-hidden="true" />
                      </template>
                      {{ $t('support.ssh') }}
                      <template #suffix>
                        <FontAwesomeIcon :icon="faChevronDown" aria-hidden="true" />
                      </template>
                    </NeButton>
                  </template>
                </NeDropdown>
                <!-- a single node: the cell keeps the width of a cluster's dropdown, so
                     the actions stay aligned while the button keeps its own size -->
                <div v-else-if="group.status === 'active' && group.system_key" class="grid">
                  <NeButton
                    kind="tertiary"
                    size="sm"
                    class="pointer-events-none invisible col-start-1 row-start-1"
                    tabindex="-1"
                    aria-hidden="true"
                  >
                    <template #prefix>
                      <FontAwesomeIcon :icon="faTerminal" />
                    </template>
                    {{ $t('support.ssh') }}
                    <template #suffix>
                      <FontAwesomeIcon :icon="faChevronDown" />
                    </template>
                  </NeButton>
                  <NeButton
                    kind="tertiary"
                    size="sm"
                    class="col-start-1 row-start-1 justify-self-start"
                    :title="$t('support.ssh_copy_hint')"
                    @click="copySSHCommand(group)"
                  >
                    <template #prefix>
                      <FontAwesomeIcon :icon="faTerminal" aria-hidden="true" />
                    </template>
                    {{ $t('support.ssh') }}
                  </NeButton>
                </div>
                <NeButton
                  v-if="group.status === 'active' || group.status === 'pending'"
                  kind="tertiary"
                  size="sm"
                  @click="handleExtendGroup(group)"
                >
                  <template #prefix>
                    <FontAwesomeIcon :icon="faClockRotateLeft" aria-hidden="true" />
                  </template>
                  {{ $t('support.extend') }}
                </NeButton>
                <NeButton
                  v-if="group.status === 'active' || group.status === 'pending'"
                  kind="tertiary"
                  size="sm"
                  :loading="closingSessionId === group.system_id"
                  :disabled="!!closingSessionId"
                  @click="handleCloseGroup(group)"
                >
                  <template #prefix>
                    <FontAwesomeIcon :icon="faXmark" aria-hidden="true" />
                  </template>
                  {{ $t('support.close') }}
                </NeButton>
                <NeButton
                  v-if="scope === 'history'"
                  kind="tertiary"
                  size="sm"
                  @click="handleShowHistory(group)"
                >
                  <template #prefix>
                    <FontAwesomeIcon :icon="faClockRotateLeft" aria-hidden="true" />
                  </template>
                  {{ $t('common.details') }}
                </NeButton>
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
                savePageSizeToStorage(supportSessionsTableId(scope), size)
              }
            "
          />
        </template>
      </NeTable>
    </div>
    <SupportExtendModal
      :group="extendGroup"
      @close="extendGroup = null"
      @extended="(expiresAt: Date) => extendGroup && onExtended(extendGroup, expiresAt)"
    />
    <SupportHistoryDrawer
      :is-shown="isHistoryShown"
      :group="historyGroup"
      @close="isHistoryShown = false"
    />
  </div>
</template>
