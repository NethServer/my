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
  faCopy,
  faCheck,
  faChevronDown,
  faChevronRight,
  faTrashCan,
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
  NeModal,
  NeTextInput,
  NeToggle,
  NeSpinner,
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
  type SessionUserCredential,
  type SessionDomainUser,
  extendSupportSession,
  closeSupportSession,
  getSupportSessionServices,
  generateSupportProxyToken,
  addSupportSessionServices,
  removeSupportSessionService,
  getSupportSessionUsers,
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

async function handleOpenService(session: SessionRef, serviceName: string, path?: string) {
  try {
    const result = await generateSupportProxyToken(session.id, serviceName)
    let baseUrl = result.url
    if (path && path !== '/') {
      baseUrl = baseUrl.replace(/\/$/, '') + path + '/'
    }
    const url = baseUrl + '?token=' + result.token
    window.open(url, '_blank')
  } catch (error) {
    console.error('Cannot generate proxy token:', error)
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

// ── Unified Services & Credentials modal ──

interface MergedCredentials {
  clusterAdmin: SessionUserCredential | null
  domainUsers: SessionDomainUser[]
  localUsers: SessionUserCredential[]
}

// A module group for the unified modal: module name + label + domain + credentials + services
interface UnifiedModuleGroup {
  moduleId: string
  moduleLabel: string
  nodeId: string
  domain: string
  password: string
  services: { name: string; session: SessionRef; host: string; path: string }[]
}

const unifiedGroup = ref<SystemSessionGroup | null>(null)
const unifiedLoading = ref(false)
const unifiedCredentials = ref<MergedCredentials | null>(null)
const unifiedModules = ref<UnifiedModuleGroup[]>([])
const unifiedUngrouped = ref<
  { name: string; label: string; session: SessionRef; host: string; path: string }[]
>([])
const copiedField = ref<string | null>(null)
const expandedModules = ref<Set<string>>(new Set())

function toggleModule(moduleId: string) {
  if (expandedModules.value.has(moduleId)) {
    expandedModules.value.delete(moduleId)
  } else {
    expandedModules.value.add(moduleId)
  }
}

async function handleOpenUnified(group: SystemSessionGroup) {
  unifiedGroup.value = group
  unifiedLoading.value = true
  unifiedCredentials.value = null
  unifiedModules.value = []
  unifiedUngrouped.value = []
  expandedModules.value = new Set()

  try {
    // Fetch fresh services and credentials from all active sessions
    const activeSessions = group.sessions.filter((s) => s.status === 'active')

    // Re-fetch services to get latest state (handles add/remove since last open)
    await Promise.all(
      activeSessions.map((s) =>
        getSupportSessionServices(s.id)
          .then((svcGroups) => {
            servicesMap.value[s.id] = svcGroups || []
          })
          .catch(() => {}),
      ),
    )

    const usersResults = await Promise.all(
      activeSessions.map((s) => getSupportSessionUsers(s.id).catch(() => null)),
    )

    // Merge credentials
    const merged: MergedCredentials = { clusterAdmin: null, domainUsers: [], localUsers: [] }
    const seenDomains = new Set<string>()
    const seenLocal = new Set<string>()
    for (const r of usersResults) {
      if (!r?.users?.users) continue
      const u = r.users.users
      if (u.cluster_admin && !merged.clusterAdmin) merged.clusterAdmin = u.cluster_admin
      for (const du of u.domain_users || []) {
        if (!seenDomains.has(du.domain)) {
          seenDomains.add(du.domain)
          merged.domainUsers.push(du)
        }
      }
      for (const lu of u.local_users || []) {
        if (!seenLocal.has(lu.username)) {
          seenLocal.add(lu.username)
          merged.localUsers.push(lu)
        }
      }
    }
    const hasData =
      merged.clusterAdmin || merged.domainUsers.length > 0 || merged.localUsers.length > 0
    unifiedCredentials.value = hasData ? merged : null

    // Build domain → password map
    const domainPasswords = new Map<string, string>()
    for (const du of merged.domainUsers) {
      domainPasswords.set(du.domain, du.password)
    }

    // Build module_domains map from all user reports
    const moduleDomains = new Map<string, string>()
    for (const r of usersResults) {
      if (!r?.users?.users?.module_domains) continue
      for (const [modId, domain] of Object.entries(r.users.users.module_domains)) {
        moduleDomains.set(modId, domain)
      }
    }

    // Build unified module groups from services
    const moduleMap = new Map<string, UnifiedModuleGroup>()
    const ungrouped: typeof unifiedUngrouped.value = []
    const seenServices = new Set<string>()

    for (const session of activeSessions) {
      for (const sg of servicesMap.value[session.id] || []) {
        for (const svc of sg.services) {
          if (seenServices.has(svc.name)) continue
          seenServices.add(svc.name)

          if (!svc.moduleId) {
            ungrouped.push({
              name: svc.name,
              label: svc.label,
              session,
              host: svc.host,
              path: svc.path,
            })
            continue
          }

          let mg = moduleMap.get(svc.moduleId)
          if (!mg) {
            const domain = moduleDomains.get(svc.moduleId) || ''
            mg = {
              moduleId: svc.moduleId,
              moduleLabel: svc.label || sg.moduleLabel || '',
              nodeId: svc.nodeId || sg.nodeId || '',
              domain,
              password: domain ? domainPasswords.get(domain) || '' : '',
              services: [],
            }
            moduleMap.set(svc.moduleId, mg)
          }
          if (!mg.moduleLabel && (svc.label || sg.moduleLabel)) {
            mg.moduleLabel = svc.label || sg.moduleLabel
          }
          mg.services.push({ name: svc.name, session, host: svc.host, path: svc.path })
        }
      }
    }

    // Sort services within each module by host (or name if no host)
    for (const mg of moduleMap.values()) {
      mg.services.sort((a, b) => {
        const ak = a.host ? `${a.host}${a.path || ''}` : a.name
        const bk = b.host ? `${b.host}${b.path || ''}` : b.name
        return ak.localeCompare(bk)
      })
    }

    // Sort modules by nodeId then moduleId
    unifiedModules.value = Array.from(moduleMap.values()).sort((a, b) => {
      const nc = (a.nodeId || '').localeCompare(b.nodeId || '', undefined, { numeric: true })
      if (nc !== 0) return nc
      return a.moduleId.localeCompare(b.moduleId)
    })
    unifiedUngrouped.value = ungrouped.sort((a, b) => a.name.localeCompare(b.name))
  } catch (error) {
    console.error('Cannot load services & credentials:', error)
  } finally {
    unifiedLoading.value = false
  }
}

function handleCloseUnified() {
  unifiedGroup.value = null
}

async function copyToClipboard(text: string, fieldId: string) {
  await navigator.clipboard.writeText(text)
  copiedField.value = fieldId
  setTimeout(() => {
    copiedField.value = null
  }, 2000)
}

// Check if unified modal has any services
function groupHasServices(group: SystemSessionGroup): boolean {
  return group.sessions.some((s) =>
    (servicesMap.value[s.id] || []).some((g) => g.services.length > 0),
  )
}

async function handleRemoveService(session: SessionRef, serviceName: string) {
  try {
    await removeSupportSessionService(session.id, serviceName)
    // Remove from local modal state immediately
    unifiedUngrouped.value = unifiedUngrouped.value.filter((s) => s.name !== serviceName)
    // Also remove from servicesMap cache so reopening the modal won't show it
    const cached = servicesMap.value[session.id]
    if (cached) {
      servicesMap.value[session.id] = cached
        .map((g) => ({ ...g, services: g.services.filter((s) => s.name !== serviceName) }))
        .filter((g) => g.services.length > 0)
    }
    // Re-fetch after delay to sync with server
    setTimeout(() => {
      getSupportSessionServices(session.id)
        .then((svcGroups) => {
          servicesMap.value[session.id] = svcGroups || []
        })
        .catch(() => {})
    }, 2000)
  } catch (error) {
    console.error('Cannot remove service:', error)
  }
}

async function handleAddService() {
  if (!addServiceGroup.value) return
  addServiceError.value = ''
  addServiceLoading.value = true
  try {
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
                <NeButton
                  v-if="group.status === 'active' && groupHasServices(group)"
                  kind="tertiary"
                  size="sm"
                  @click="handleOpenUnified(group)"
                >
                  <template #prefix>
                    <FontAwesomeIcon :icon="faUpRightFromSquare" aria-hidden="true" />
                  </template>
                  {{ $t('support.services') }}
                </NeButton>
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
    <!-- Unified Services & Credentials modal -->
    <NeModal
      v-if="unifiedGroup"
      :visible="!!unifiedGroup"
      :title="$t('support.services_and_credentials')"
      kind="info"
      :cancel-label="$t('common.close')"
      :close-aria-label="$t('common.close')"
      size="xxl"
      class="hide-primary-button"
      @close="handleCloseUnified"
    >
      <div class="-mr-2 flex max-h-[65vh] flex-col gap-3 overflow-y-auto pr-2">
        <div v-if="unifiedLoading" class="flex justify-center py-4">
          <NeSpinner />
        </div>
        <template v-else>
          <!-- Cluster Admin credentials -->
          <div
            v-if="unifiedCredentials?.clusterAdmin"
            class="shrink-0 rounded-md border border-gray-200 p-3 dark:border-gray-700"
          >
            <h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
              {{ $t('support.cluster_admin') }}
            </h4>
            <div class="flex flex-col gap-1 text-sm">
              <div class="flex items-center justify-between">
                <span class="text-gray-500 dark:text-gray-400">{{ $t('support.username') }}:</span>
                <div class="flex items-center gap-1">
                  <code class="text-gray-900 dark:text-gray-100">{{
                    unifiedCredentials.clusterAdmin.username
                  }}</code>
                  <button
                    class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                    @click="copyToClipboard(unifiedCredentials.clusterAdmin!.username, 'ca-user')"
                  >
                    <FontAwesomeIcon
                      :icon="copiedField === 'ca-user' ? faCheck : faCopy"
                      class="h-3 w-3"
                    />
                  </button>
                </div>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-gray-500 dark:text-gray-400">{{ $t('support.password') }}:</span>
                <div class="flex items-center gap-1">
                  <code class="text-gray-900 dark:text-gray-100">{{
                    unifiedCredentials.clusterAdmin.password
                  }}</code>
                  <button
                    class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                    @click="copyToClipboard(unifiedCredentials.clusterAdmin!.password, 'ca-pass')"
                  >
                    <FontAwesomeIcon
                      :icon="copiedField === 'ca-pass' ? faCheck : faCopy"
                      class="h-3 w-3"
                    />
                  </button>
                </div>
              </div>
            </div>
            <!-- Open cluster-admin link -->
            <button
              v-if="unifiedUngrouped.find((s) => s.name === 'cluster-admin')"
              class="text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 mt-2 flex items-center gap-1.5 text-sm"
              @click="
                handleOpenService(
                  unifiedUngrouped.find((s) => s.name === 'cluster-admin')!.session,
                  'cluster-admin',
                )
              "
            >
              <FontAwesomeIcon :icon="faUpRightFromSquare" class="h-3 w-3" />
              {{ $t('support.open_service') }}
            </button>
          </div>
          <!-- Local Users (NethSecurity) -->
          <div
            v-if="unifiedCredentials?.localUsers.length"
            class="shrink-0 rounded-md border border-gray-200 p-3 dark:border-gray-700"
          >
            <h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
              {{ $t('support.local_users') }}
            </h4>
            <div
              v-for="(lu, idx) in unifiedCredentials.localUsers"
              :key="idx"
              class="flex flex-col gap-1 text-sm"
            >
              <div class="flex items-center justify-between">
                <span class="text-gray-500 dark:text-gray-400">{{ $t('support.username') }}:</span>
                <div class="flex items-center gap-1">
                  <code class="text-gray-900 dark:text-gray-100">{{ lu.username }}</code>
                  <button
                    class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                    @click="copyToClipboard(lu.username, `lu-user-${idx}`)"
                  >
                    <FontAwesomeIcon
                      :icon="copiedField === `lu-user-${idx}` ? faCheck : faCopy"
                      class="h-3 w-3"
                    />
                  </button>
                </div>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-gray-500 dark:text-gray-400">{{ $t('support.password') }}:</span>
                <div class="flex items-center gap-1">
                  <code class="text-gray-900 dark:text-gray-100">{{ lu.password }}</code>
                  <button
                    class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                    @click="copyToClipboard(lu.password, `lu-pass-${idx}`)"
                  >
                    <FontAwesomeIcon
                      :icon="copiedField === `lu-pass-${idx}` ? faCheck : faCopy"
                      class="h-3 w-3"
                    />
                  </button>
                </div>
              </div>
            </div>
            <!-- Open nethsecurity-ui link -->
            <button
              v-if="unifiedUngrouped.find((s) => s.name === 'nethsecurity-ui')"
              class="text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 mt-2 flex items-center gap-1.5 text-sm"
              @click="
                handleOpenService(
                  unifiedUngrouped.find((s) => s.name === 'nethsecurity-ui')!.session,
                  'nethsecurity-ui',
                )
              "
            >
              <FontAwesomeIcon :icon="faUpRightFromSquare" class="h-3 w-3" />
              {{ $t('support.open_service') }}
            </button>
          </div>
          <!-- Module groups with services + domain credentials (collapsible) -->
          <template v-for="(mg, mgIdx) in unifiedModules" :key="mg.moduleId">
            <div
              class="shrink-0 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-gray-500 dark:bg-gray-800"
            >
              <!-- Clickable module header -->
              <button
                class="flex w-full items-center gap-2 px-4 py-4 text-left hover:bg-gray-50 dark:hover:bg-gray-800"
                @click="toggleModule(mg.moduleId)"
              >
                <FontAwesomeIcon
                  :icon="expandedModules.has(mg.moduleId) ? faChevronDown : faChevronRight"
                  class="h-3.5 w-3.5 shrink-0 text-gray-500 dark:text-gray-400"
                />
                <h4 class="flex-1 text-sm font-semibold text-gray-900 dark:text-gray-100">
                  {{ mg.moduleId }}
                  <span v-if="mg.moduleLabel" class="font-normal text-gray-500 dark:text-gray-400">
                    ({{ mg.moduleLabel }})
                  </span>
                </h4>
                <NeBadge v-if="mg.nodeId" :text="`Node ${mg.nodeId}`" kind="secondary" size="sm" />
                <NeBadge :text="`${mg.services.length}`" kind="secondary" size="sm" />
              </button>
              <!-- Expanded content -->
              <div
                v-if="expandedModules.has(mg.moduleId)"
                class="border-t border-gray-200 p-3 dark:border-gray-700"
              >
                <!-- Domain credentials for this module -->
                <div
                  v-if="mg.domain && mg.password"
                  class="mb-2 rounded bg-gray-50 p-2 text-sm dark:bg-gray-800/50"
                >
                  <div class="flex items-center justify-between">
                    <span class="text-gray-500 dark:text-gray-400"
                      >{{ $t('support.domain') }}:</span
                    >
                    <span class="text-gray-700 dark:text-gray-300">{{ mg.domain }}</span>
                  </div>
                  <div class="flex items-center justify-between">
                    <span class="text-gray-500 dark:text-gray-400"
                      >{{ $t('support.username') }}:</span
                    >
                    <div class="flex items-center gap-1">
                      <code class="text-gray-900 dark:text-gray-100">{{
                        unifiedCredentials?.domainUsers[0]?.username
                      }}</code>
                      <button
                        class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                        @click.stop="
                          copyToClipboard(
                            unifiedCredentials?.domainUsers[0]?.username || '',
                            `mg-user-${mgIdx}`,
                          )
                        "
                      >
                        <FontAwesomeIcon
                          :icon="copiedField === `mg-user-${mgIdx}` ? faCheck : faCopy"
                          class="h-3 w-3"
                        />
                      </button>
                    </div>
                  </div>
                  <div class="flex items-center justify-between">
                    <span class="text-gray-500 dark:text-gray-400"
                      >{{ $t('support.password') }}:</span
                    >
                    <div class="flex items-center gap-1">
                      <code class="text-gray-900 dark:text-gray-100">{{ mg.password }}</code>
                      <button
                        class="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                        @click.stop="copyToClipboard(mg.password, `mg-pass-${mgIdx}`)"
                      >
                        <FontAwesomeIcon
                          :icon="copiedField === `mg-pass-${mgIdx}` ? faCheck : faCopy"
                          class="h-3 w-3"
                        />
                      </button>
                    </div>
                  </div>
                </div>
                <!-- Service links -->
                <div class="flex flex-col gap-0.5">
                  <button
                    v-for="svc in mg.services"
                    :key="svc.name"
                    class="text-primary-600 dark:text-primary-400 flex items-center gap-1.5 rounded px-1 py-0.5 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-800"
                    @click="handleOpenService(svc.session, svc.name, svc.path)"
                  >
                    <FontAwesomeIcon :icon="faUpRightFromSquare" class="h-3 w-3 shrink-0" />
                    <span v-if="svc.host" class="min-w-0 truncate"
                      >{{ svc.host }}{{ svc.path || '' }}</span
                    >
                    <span v-else class="min-w-0 truncate">{{ svc.name }}</span>
                    <span class="min-w-0 shrink truncate text-gray-400"
                      >({{ svc.name }})</span
                    >
                  </button>
                </div>
              </div>
            </div>
          </template>
          <!-- Ungrouped services (no moduleId, excluding cluster-admin) -->
          <div
            v-if="
              unifiedUngrouped.filter(
                (s) => s.name !== 'cluster-admin' && s.name !== 'nethsecurity-ui',
              ).length
            "
            class="shrink-0 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-gray-500 dark:bg-gray-800"
          >
            <button
              class="flex w-full items-center gap-2 px-4 py-4 text-left hover:bg-gray-50 dark:hover:bg-gray-800"
              @click="toggleModule('__ungrouped__')"
            >
              <FontAwesomeIcon
                :icon="expandedModules.has('__ungrouped__') ? faChevronDown : faChevronRight"
                class="h-3 w-3 shrink-0 text-gray-400"
              />
              <h4 class="flex-1 text-sm font-semibold text-gray-900 dark:text-gray-100">
                {{ $t('support.custom_services') }}
              </h4>
              <NeBadge
                :text="`${unifiedUngrouped.filter((s) => s.name !== 'cluster-admin' && s.name !== 'nethsecurity-ui').length}`"
                kind="secondary"
                size="sm"
              />
            </button>
            <div
              v-if="expandedModules.has('__ungrouped__')"
              class="border-t border-gray-200 p-3 dark:border-gray-700"
            >
              <div class="flex flex-col gap-1">
                <div
                  v-for="svc in unifiedUngrouped.filter(
                    (s) => s.name !== 'cluster-admin' && s.name !== 'nethsecurity-ui',
                  )"
                  :key="svc.name"
                  class="flex items-center justify-between rounded px-1 py-0.5 text-sm"
                >
                  <button
                    class="text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 flex items-center gap-1.5"
                    @click="handleOpenService(svc.session, svc.name, svc.path)"
                  >
                    <FontAwesomeIcon :icon="faUpRightFromSquare" class="h-3 w-3 shrink-0" />
                    <span>{{ svc.label || svc.name }}</span>
                    <span v-if="svc.label" class="text-xs text-gray-400">({{ svc.name }})</span>
                  </button>
                  <button
                    class="p-1 text-gray-400 hover:text-red-500 dark:hover:text-red-400"
                    @click="handleRemoveService(svc.session, svc.name)"
                  >
                    <FontAwesomeIcon :icon="faTrashCan" class="h-3 w-3" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </template>
      </div>
    </NeModal>
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

<style>
.hide-primary-button button[type='submit'] {
  display: none;
}
</style>
