<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  faHeadset,
  faClockRotateLeft,
  faXmark,
  faTerminal,
  faUpRightFromSquare,
  faPlug,
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
  NeEmptyState,
  NeInlineNotification,
  NeBadge,
  NeDropdown,
  NeModal,
  NeTextInput,
  NeToggle,
  type NeDropdownItem,
} from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { savePageSizeToStorage } from '@/lib/tablePageSize'
import { useSupportSessions } from '@/queries/support/supportSessions'
import {
  SUPPORT_SESSIONS_KEY,
  SUPPORT_SESSIONS_TABLE_ID,
  type SessionRef,
  type SupportSessionStatus,
  type SystemSessionGroup,
  type SupportServiceGroup,
  extendSupportSession,
  closeSupportSession,
  getSupportSessionServices,
  generateSupportProxyToken,
  addSupportSessionServices,
} from '@/lib/support/support'
import UpdatingSpinner from '@/components/UpdatingSpinner.vue'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import { canConnectSystems } from '@/lib/permissions'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { useQueryCache } from '@pinia/colada'
import { getProductName } from '@/lib/systems/systems'
import SupportTerminal from '@/components/support/SupportTerminal.vue'

const { locale, t } = useI18n()
const queryCache = useQueryCache()
const { state, asyncStatus, pageNum, pageSize } = useSupportSessions()

const sessionGroups = computed(() => {
  return state.value.data?.support_sessions || []
})

const pagination = computed(() => {
  return state.value.data?.pagination
})

const isNoDataEmptyStateShown = computed(() => {
  return !sessionGroups.value?.length && state.value.status === 'success'
})

const extendingSessionId = ref<string | null>(null)
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

async function handleExtendGroup(group: SystemSessionGroup) {
  extendingSessionId.value = group.system_id
  try {
    await Promise.all(group.sessions.map((s) => extendSupportSession(s.id, 24)))
    queryCache.invalidateQueries({ key: [SUPPORT_SESSIONS_KEY] })
  } catch (error) {
    console.error('Cannot extend support session:', error)
  } finally {
    extendingSessionId.value = null
  }
}

const terminalGroup = ref<SystemSessionGroup | null>(null)

function handleOpenTerminal(group: SystemSessionGroup) {
  terminalGroup.value = group
}

function handleCloseTerminal() {
  terminalGroup.value = null
}

// Services map keyed by session ID (individual sessions, not groups)
const servicesMap = ref<Record<string, SupportServiceGroup[]>>({})
const openingServiceId = ref<string | null>(null)

watch(
  sessionGroups,
  (groups) => {
    for (const group of groups) {
      for (const session of group.sessions) {
        if (session.status === 'active' && !servicesMap.value[session.id]) {
          getSupportSessionServices(session.id)
            .then((svcGroups) => {
              servicesMap.value[session.id] = svcGroups || []
            })
            .catch(() => {
              servicesMap.value[session.id] = []
            })
        }
      }
    }
  },
  { immediate: true },
)

function groupHasServices(group: SystemSessionGroup): boolean {
  return group.sessions.some((s) =>
    (servicesMap.value[s.id] || []).some((g) => g.services.length > 0),
  )
}

function getGroupServiceDropdownItems(group: SystemSessionGroup): NeDropdownItem[] {
  const items: NeDropdownItem[] = []

  // Collect all service groups from all sessions, deduplicating by service name.
  interface ServiceEntry {
    group: SupportServiceGroup
    session: SessionRef
  }
  const allGroups: ServiceEntry[] = []
  const seenServices = new Set<string>()

  for (const session of group.sessions) {
    for (const sg of servicesMap.value[session.id] || []) {
      const deduped: SupportServiceGroup = { ...sg, services: [] }
      for (const svc of sg.services) {
        if (seenServices.has(svc.name)) continue
        seenServices.add(svc.name)
        deduped.services.push(svc)
      }
      if (deduped.services.length > 0) {
        allGroups.push({ group: deduped, session })
      }
    }
  }

  // Sort by nodeId then moduleId
  allGroups.sort((a, b) => {
    const nodeA = a.group.nodeId || ''
    const nodeB = b.group.nodeId || ''
    const nc = nodeA.localeCompare(nodeB, undefined, { numeric: true })
    if (nc !== 0) return nc
    return a.group.moduleId.localeCompare(b.group.moduleId)
  })

  // Check if services span multiple nodes
  const nodeIds = new Set(allGroups.map((e) => e.group.nodeId).filter(Boolean))
  const multiNode = nodeIds.size > 1
  let lastNodeId = ''

  for (const entry of allGroups) {
    const { group: sg, session } = entry

    // Node header
    if (multiNode && sg.nodeId && sg.nodeId !== lastNodeId) {
      items.push({
        id: `node-${sg.nodeId}`,
        label: `— Node ${sg.nodeId} —`,
        disabled: true,
      })
      lastNodeId = sg.nodeId
    }

    // Module header
    if (sg.moduleId) {
      const header = sg.moduleLabel ? `${sg.moduleId} (${sg.moduleLabel})` : sg.moduleId
      items.push({
        id: `header-${sg.nodeId}-${sg.moduleId}`,
        label: header,
        disabled: true,
      })
    }

    // Service items
    for (const svc of sg.services) {
      let label = svc.name
      if (svc.host || svc.path) {
        const hostPath = (svc.host || '') + (svc.path || '')
        label += ` (${hostPath})`
      }
      items.push({
        id: `${session.id}-${svc.name}`,
        label,
        icon: faUpRightFromSquare,
        action: () => handleOpenService(session, svc.name, svc.path),
      })
    }
  }

  return items
}

async function handleOpenService(session: SessionRef, serviceName: string, path?: string) {
  openingServiceId.value = session.id
  try {
    const result = await generateSupportProxyToken(session.id, serviceName)
    // Append the route path (e.g., /pbx-report) so Traefik matches the correct route
    let baseUrl = result.url
    if (path && path !== '/') {
      // result.url ends with '/', path starts with '/'
      baseUrl = baseUrl.replace(/\/$/, '') + path + '/'
    }
    const url = baseUrl + '?token=' + result.token
    window.open(url, '_blank')
  } catch (error) {
    console.error('Cannot generate proxy token:', error)
  } finally {
    openingServiceId.value = null
  }
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

// Add service modal
const addServiceGroup = ref<SystemSessionGroup | null>(null)
const addServiceName = ref('')
const addServiceTarget = ref('')
const addServiceLabel = ref('')
const addServiceTls = ref(false)
const addServiceLoading = ref(false)
const addServiceError = ref('')

function handleOpenAddService(group: SystemSessionGroup) {
  addServiceGroup.value = group
  addServiceName.value = ''
  addServiceTarget.value = ''
  addServiceLabel.value = ''
  addServiceTls.value = false
  addServiceError.value = ''
}

function handleCloseAddService() {
  addServiceGroup.value = null
}

async function handleAddService() {
  if (!addServiceGroup.value) return
  addServiceError.value = ''
  addServiceLoading.value = true
  try {
    // Use the first active session id as the target
    const activeSession = addServiceGroup.value.sessions.find((s) => s.status === 'active')
    if (!activeSession) {
      addServiceError.value = t('support.no_active_session')
      return
    }
    await addSupportSessionServices(activeSession.id, [
      {
        name: addServiceName.value,
        target: addServiceTarget.value,
        label: addServiceLabel.value,
        tls: addServiceTls.value,
      },
    ])
    // Wait for the Redis → support service → tunnel-client → manifest round-trip
    // before re-fetching, otherwise the GET arrives before the manifest is updated
    handleCloseAddService()
    setTimeout(() => {
      getSupportSessionServices(activeSession.id)
        .then((svcGroups) => {
          servicesMap.value[activeSession.id] = svcGroups || []
        })
        .catch(() => {})
    }, 1500)
  } catch (error: unknown) {
    const axiosError = error as { response?: { data?: { message?: string } } }
    addServiceError.value = axiosError?.response?.data?.message || t('support.add_service_error')
  } finally {
    addServiceLoading.value = false
  }
}
</script>

<template>
  <div>
    <!-- update indicator -->
    <UpdatingSpinner v-if="asyncStatus === 'loading' && state.status !== 'pending'" />
    <div class="flex flex-col gap-6">
      <!-- error notification -->
      <NeInlineNotification
        v-if="state.status === 'error'"
        kind="error"
        :title="$t('support.cannot_retrieve_sessions')"
        :description="state.error.message"
      />
      <!-- no data empty state -->
      <NeEmptyState
        v-if="isNoDataEmptyStateShown"
        :title="$t('support.no_sessions')"
        :description="$t('support.no_sessions_description')"
        :icon="faHeadset"
        class="bg-white dark:bg-gray-950"
      />
      <NeTable
        v-else
        :aria-label="$t('support.title')"
        card-breakpoint="xl"
        :loading="state.status === 'pending'"
        :skeleton-columns="7"
        :skeleton-rows="5"
      >
        <NeTableHead>
          <NeTableHeadCell>{{ $t('support.system') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('support.type') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('support.organization') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('support.started_at') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('support.expires_at') }}</NeTableHeadCell>
          <NeTableHeadCell>{{ $t('support.status') }}</NeTableHeadCell>
          <NeTableHeadCell v-if="canConnectSystems()">
            <!-- no header for actions -->
          </NeTableHeadCell>
        </NeTableHead>
        <NeTableBody>
          <NeTableRow v-for="group in sessionGroups" :key="group.system_id">
            <NeTableCell :data-label="$t('support.system')">
              <div class="flex items-center gap-2">
                <span>{{ group.system_name || group.system_key }}</span>
                <NeBadge
                  v-if="group.node_count > 1"
                  :text="`${group.node_count} nodes`"
                  kind="secondary"
                  size="sm"
                />
              </div>
            </NeTableCell>
            <NeTableCell :data-label="$t('support.type')">
              {{ group.system_type ? getProductName(group.system_type) : '-' }}
            </NeTableCell>
            <NeTableCell :data-label="$t('support.organization')">
              {{ group.organization?.name || '-' }}
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
              <div class="-ml-2.5 flex gap-2 xl:ml-0 xl:justify-end">
                <NeButton
                  v-if="group.status === 'active'"
                  kind="tertiary"
                  size="sm"
                  @click="handleOpenTerminal(group)"
                >
                  <template #prefix>
                    <FontAwesomeIcon :icon="faTerminal" aria-hidden="true" />
                  </template>
                  {{ $t('support.terminal') }}
                </NeButton>
                <NeDropdown
                  v-if="group.status === 'active' && groupHasServices(group)"
                  :items="getGroupServiceDropdownItems(group)"
                  :align-to-right="true"
                  menu-classes="max-h-80 overflow-y-auto"
                >
                  <template #button>
                    <NeButton
                      kind="tertiary"
                      size="sm"
                      :loading="openingServiceId === group.system_id"
                    >
                      <template #prefix>
                        <FontAwesomeIcon :icon="faUpRightFromSquare" aria-hidden="true" />
                      </template>
                      {{ $t('support.open_service') }}
                    </NeButton>
                  </template>
                </NeDropdown>
                <NeButton
                  v-if="group.status === 'active'"
                  kind="tertiary"
                  size="sm"
                  @click="handleOpenAddService(group)"
                >
                  <template #prefix>
                    <FontAwesomeIcon :icon="faPlug" aria-hidden="true" />
                  </template>
                  {{ $t('support.add_service') }}
                </NeButton>
                <NeButton
                  v-if="group.status === 'active' || group.status === 'pending'"
                  kind="tertiary"
                  size="sm"
                  :loading="extendingSessionId === group.system_id"
                  :disabled="!!extendingSessionId"
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
                savePageSizeToStorage(SUPPORT_SESSIONS_TABLE_ID, size)
              }
            "
          />
        </template>
      </NeTable>
    </div>
    <!-- Terminal modal -->
    <SupportTerminal
      v-if="terminalGroup"
      :sessions="terminalGroup.sessions"
      :system-name="terminalGroup.system_name || terminalGroup.system_key"
      @close="handleCloseTerminal"
    />
    <!-- Add service modal -->
    <NeModal
      v-if="addServiceGroup"
      :visible="!!addServiceGroup"
      :title="$t('support.add_service')"
      :primary-label="$t('support.add_service')"
      :cancel-label="$t('common.cancel')"
      :primary-button-loading="addServiceLoading"
      :close-aria-label="$t('common.close')"
      @close="handleCloseAddService"
      @primary-click="handleAddService"
    >
      <div class="flex flex-col gap-4">
        <NeInlineNotification
          v-if="addServiceError"
          kind="error"
          :title="$t('support.add_service_error')"
          :description="addServiceError"
        />
        <NeTextInput
          v-model="addServiceName"
          :label="$t('support.service_name')"
          :placeholder="$t('support.service_name_placeholder')"
          :helper-text="$t('support.service_name_helper')"
        />
        <NeTextInput
          v-model="addServiceTarget"
          :label="$t('support.service_target')"
          :placeholder="$t('support.service_target_placeholder')"
          :helper-text="$t('support.service_target_helper')"
        />
        <NeTextInput
          v-model="addServiceLabel"
          :label="$t('support.service_label')"
          :placeholder="$t('support.service_label_placeholder')"
        />
        <NeToggle v-model="addServiceTls" :label="$t('support.service_tls')" />
      </div>
    </NeModal>
  </div>
</template>
