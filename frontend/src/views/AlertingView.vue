<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeBadgeV2,
  NeButton,
  NeCard,
  NeCombobox,
  NeDropdownFilter,
  NeEmptyState,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
  NeTabs,
  NeTextArea,
  NeTextInput,
  type FilterOption,
  type NeBadgeV2Kind,
  type NeComboboxOption,
  type Tab,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  faBell,
  faArrowsRotate,
  faExclamationTriangle,
  faCog,
  faCheck,
  faCopy,
  faXmark,
  faTrash,
  faPenToSquare,
} from '@fortawesome/free-solid-svg-icons'
import { useTabs } from '@/composables/useTabs'
import { useLoginStore } from '@/stores/login'
import { useAlerts } from '@/queries/alerting/alerts'
import {
  getAlertDescription,
  getAlertSummary,
  getAlertingConfig,
  postAlertingConfig,
  deleteAlertingConfig,
  type Alert,
  type AlertingConfig,
} from '@/lib/alerting'
import { useI18n } from 'vue-i18n'
import { computed, ref, watch } from 'vue'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import DataItem from '@/components/DataItem.vue'
import UpdatingSpinner from '@/components/UpdatingSpinner.vue'
import { useNotificationsStore } from '@/stores/notifications'
import { useQuery } from '@pinia/colada'
import { organizationsQuery } from '@/queries/organizations/organizations'
import { getOrganizationIcon } from '@/lib/organizations/organizations'

const { t, locale } = useI18n()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()

// ── Organisation selector ──────────────────────────────────────────────────────
const { state: orgsState } = useQuery({
  ...organizationsQuery,
  enabled: () => !!loginStore.jwtToken,
})

const organizationOptions = computed((): NeComboboxOption[] => {
  return (orgsState.value.data ?? [])
    .filter((o) => o.type.toLowerCase() !== 'owner')
    .map((o) => ({
      id: o.logto_id,
      label: o.name,
      description: o.type,
      icon: getOrganizationIcon(o.type),
    }))
})

const selectedOrgId = ref('')

// ── Page tabs (Alerts / Configuration) ────────────────────────────────────────
const { tabs: pageTabs, selectedTab: selectedPageTab } = useTabs([
  { name: 'alerts', label: t('alerting.title') },
  { name: 'config', label: t('alerting.config_title') },
])

// ── Alerts ─────────────────────────────────────────────────────────────────────
const {
  state: alertsState,
  asyncStatus: alertsAsyncStatus,
  refetch: refetchAlerts,
  organizationId: alertsOrgId,
  stateFilter,
  severityFilter,
  systemKeyFilter,
  resetFilters,
  areDefaultFiltersApplied,
} = useAlerts()

const isOrganizationsLoading = computed(
  () => loginStore.loadingUserInfo || orgsState.value.status === 'pending',
)
const hasSelectableOrganizations = computed(() => organizationOptions.value.length > 0)
const hasSelectedOrganization = computed(() => !!selectedOrgId.value)
const activeAlertsCount = computed(() => alertsState.value.data?.length ?? 0)
const activeAlertsCountLabel = computed(() =>
  t('alerting.active_alerts_count', { num: activeAlertsCount.value }, activeAlertsCount.value),
)

const systemKeyInput = ref('')

const stateFilterOptions = ref<FilterOption[]>([
  { id: 'active', label: t('alerting.state_active') },
  { id: 'suppressed', label: t('alerting.state_suppressed') },
  { id: 'unprocessed', label: t('alerting.state_unprocessed') },
])

const severityFilterOptions = ref<FilterOption[]>([
  { id: 'critical', label: t('alerting.severity_critical') },
  { id: 'warning', label: t('alerting.severity_warning') },
  { id: 'info', label: t('alerting.severity_info') },
])

const hasActiveFilters = computed(() => !areDefaultFiltersApplied())

function getAlertSummaryText(alert: Alert) {
  return getAlertSummary(alert, locale.value)
}

function getAlertDescriptionText(alert: Alert) {
  const description = getAlertDescription(alert, locale.value)
  return description !== getAlertSummaryText(alert) ? description : ''
}

function applySystemKeyFilter() {
  systemKeyFilter.value = systemKeyInput.value.trim()
}

function clearAllFilters() {
  systemKeyInput.value = ''
  resetFilters()
}

function getSeverityBadgeKind(severity: string | undefined): NeBadgeV2Kind {
  switch (severity?.toLowerCase()) {
    case 'critical':
      return 'rose'
    case 'warning':
      return 'amber'
    case 'info':
      return 'blue'
    default:
      return 'gray'
  }
}

function getStateBadgeKind(stateVal: string | undefined): NeBadgeV2Kind {
  switch (stateVal?.toLowerCase()) {
    case 'active':
      return 'rose'
    case 'suppressed':
      return 'gray'
    default:
      return 'gray'
  }
}

// ── Config ─────────────────────────────────────────────────────────────────────
const configTabs = ref<Tab[]>([
  { name: 'structured', label: t('alerting.structured') },
  { name: 'yaml', label: t('alerting.raw_yaml') },
])
const selectedConfigTab = ref('structured')

const config = ref<AlertingConfig | null>(null)
const rawYaml = ref<string>('')
const isLoading = ref(false)
const fetchError = ref<string | null>(null)
const isSaving = ref(false)
const isDisabling = ref(false)
const saveError = ref<string | null>(null)
const disableError = ref<string | null>(null)
const isEditMode = ref(false)
const editJson = ref('')
const editJsonError = ref<string | null>(null)
const isConfirmDisable = ref(false)
let configRequestId = 0

watch(
  organizationOptions,
  (options) => {
    if (!options.length) {
      selectedOrgId.value = ''
      return
    }

    if (!options.some(({ id }) => id === selectedOrgId.value)) {
      selectedOrgId.value = options[0].id
    }
  },
  { immediate: true },
)

watch(
  selectedOrgId,
  (id) => {
    alertsOrgId.value = id
  },
  { immediate: true },
)

async function fetchConfig(orgId: string) {
  const requestId = ++configRequestId

  if (!orgId) {
    config.value = null
    rawYaml.value = ''
    fetchError.value = null
    isLoading.value = false
    return
  }

  isLoading.value = true
  fetchError.value = null
  config.value = null
  rawYaml.value = ''

  try {
    const [jsonCfg, yamlCfg] = await Promise.all([
      getAlertingConfig(orgId),
      getAlertingConfig(orgId, 'yaml'),
    ])
    if (requestId !== configRequestId) {
      return
    }

    config.value = jsonCfg as AlertingConfig
    rawYaml.value = (yamlCfg as string) || ''
  } catch (e: unknown) {
    if (requestId !== configRequestId) {
      return
    }

    fetchError.value = e instanceof Error ? e.message : String(e)
  } finally {
    if (requestId === configRequestId) {
      isLoading.value = false
    }
  }
}

watch(
  selectedOrgId,
  (id) => {
    isEditMode.value = false
    isConfirmDisable.value = false
    fetchConfig(id)
  },
  { immediate: true },
)

const yamlCopied = ref(false)
async function copyYaml() {
  await navigator.clipboard.writeText(rawYaml.value)
  yamlCopied.value = true
  setTimeout(() => (yamlCopied.value = false), 2000)
}

function startEdit() {
  editJson.value = JSON.stringify(config.value, null, 2)
  editJsonError.value = null
  isEditMode.value = true
}

function cancelEdit() {
  isEditMode.value = false
  editJsonError.value = null
}

async function saveConfig() {
  editJsonError.value = null
  let parsed: AlertingConfig
  try {
    parsed = JSON.parse(editJson.value)
  } catch {
    editJsonError.value = 'Invalid JSON'
    return
  }

  isSaving.value = true
  saveError.value = null
  try {
    await postAlertingConfig(selectedOrgId.value, parsed)
    isEditMode.value = false
    notificationsStore.createNotification({
      kind: 'success',
      title: t('alerting.config_saved'),
      description: t('alerting.config_saved_description'),
    })
    await fetchConfig(selectedOrgId.value)
  } catch (e: unknown) {
    saveError.value = e instanceof Error ? e.message : String(e)
  } finally {
    isSaving.value = false
  }
}

async function disableAlerts() {
  isDisabling.value = true
  disableError.value = null
  isConfirmDisable.value = false
  try {
    await deleteAlertingConfig(selectedOrgId.value)
    notificationsStore.createNotification({
      kind: 'success',
      title: t('alerting.alerts_disabled'),
      description: t('alerting.alerts_disabled_description'),
    })
    await fetchConfig(selectedOrgId.value)
  } catch (e: unknown) {
    disableError.value = e instanceof Error ? e.message : String(e)
  } finally {
    isDisabling.value = false
  }
}

function formatMailAddresses(addresses: string[] | undefined) {
  if (!addresses || addresses.length === 0) return '-'
  return addresses.join(', ')
}

// ── Owner-only guard ───────────────────────────────────────────────────────────
const isOwner = computed(() => loginStore.isOwner)
</script>

<template>
  <div>
    <!-- access denied -->
    <NeInlineNotification
      v-if="!isOwner"
      kind="error"
      :title="$t('common.access_denied')"
      :description="$t('common.this_section_is_accessible_only_by_owner')"
    />

    <template v-else>
      <!-- header -->
      <div class="mb-6 flex items-start justify-between gap-4">
        <div>
          <div class="mb-1 flex items-center gap-3">
            <NeHeading tag="h3">{{ $t('alerting.alerting_title') }}</NeHeading>
            <NeBadgeV2 kind="amber" size="sm">ALPHA</NeBadgeV2>
          </div>
          <p class="max-w-2xl text-sm text-gray-500 dark:text-gray-400">
            {{ $t('alerting.alerting_page_description') }}
          </p>
          <NeInlineNotification
            kind="warning"
            :title="$t('alerting.alpha_notice')"
            class="mt-3 max-w-2xl"
          />
        </div>
      </div>

      <NeInlineNotification
        v-if="orgsState.status === 'error'"
        kind="error"
        :title="$t('alerting.cannot_retrieve_organizations')"
        :description="orgsState.error?.message"
        class="mb-6"
      />

      <NeSkeleton v-else-if="isOrganizationsLoading" :lines="8" />

      <NeEmptyState
        v-else-if="!hasSelectableOrganizations"
        :title="$t('organizations.no_organizations')"
        :description="$t('alerting.no_organizations_description')"
        :icon="faBell"
      />

      <template v-else>
        <!-- org selector -->
        <div class="mb-6 max-w-sm">
          <NeCombobox
            v-model="selectedOrgId"
            :options="organizationOptions"
            :label="$t('alerting.select_organization')"
            :placeholder="$t('organizations.choose_organization')"
            :no-results-label="$t('ne_combobox.no_results')"
            :limited-options-label="$t('ne_combobox.limited_options_label')"
            :no-options-label="$t('organizations.no_organizations')"
            :selected-label="$t('ne_combobox.selected')"
            :user-input-label="$t('ne_combobox.user_input_label')"
            :optional-label="$t('common.optional')"
          />
          <p
            v-if="hasSelectedOrganization && alertsState.status === 'success'"
            class="mt-2 text-sm text-gray-500 dark:text-gray-400"
          >
            {{ activeAlertsCountLabel }}
          </p>
        </div>

        <!-- page tabs -->
        <NeTabs
          :tabs="pageTabs"
          :selected="selectedPageTab"
          :sr-tabs-label="$t('ne_tabs.tabs')"
          :sr-select-tab-label="$t('ne_tabs.select_a_tab')"
          class="mb-6"
          @select-tab="selectedPageTab = $event"
        />

        <!-- ══ ALERTS TAB ══════════════════════════════════════════════════════ -->
        <template v-if="selectedPageTab === 'alerts'">
          <!-- toolbar -->
          <div class="mb-4 flex items-center justify-between gap-4">
            <div class="flex flex-wrap items-center gap-3">
              <NeDropdownFilter
                v-model="stateFilter"
                kind="checkbox"
                :label="$t('alerting.filter_by_state')"
                :options="stateFilterOptions"
                :clear-filter-label="$t('ne_dropdown_filter.clear_filter')"
                :open-menu-aria-label="$t('ne_dropdown_filter.open_filter')"
                :no-options-label="$t('ne_dropdown_filter.no_options')"
                :more-options-hidden-label="$t('ne_dropdown_filter.more_options_hidden')"
                :clear-search-label="$t('ne_dropdown_filter.clear_search')"
              />
              <NeDropdownFilter
                v-model="severityFilter"
                kind="checkbox"
                :label="$t('alerting.filter_by_severity')"
                :options="severityFilterOptions"
                :clear-filter-label="$t('ne_dropdown_filter.clear_filter')"
                :open-menu-aria-label="$t('ne_dropdown_filter.open_filter')"
                :no-options-label="$t('ne_dropdown_filter.no_options')"
                :more-options-hidden-label="$t('ne_dropdown_filter.more_options_hidden')"
                :clear-search-label="$t('ne_dropdown_filter.clear_search')"
              />
              <div class="flex items-center gap-2">
                <NeTextInput
                  v-model="systemKeyInput"
                  :placeholder="$t('alerting.filter_by_system_key')"
                  class="max-w-xs"
                  @keyup.enter="applySystemKeyFilter"
                />
                <NeButton kind="secondary" size="sm" @click="applySystemKeyFilter">
                  {{ $t('common.search') }}
                </NeButton>
              </div>
              <NeButton v-if="hasActiveFilters" kind="tertiary" size="sm" @click="clearAllFilters">
                {{ $t('common.reset_filters') }}
              </NeButton>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <UpdatingSpinner
                v-if="alertsAsyncStatus === 'loading' && alertsState.status !== 'pending'"
              />
              <NeButton kind="tertiary" size="sm" @click="refetchAlerts()">
                <template #prefix>
                  <FontAwesomeIcon :icon="faArrowsRotate" />
                </template>
              </NeButton>
            </div>
          </div>

          <!-- error -->
          <NeInlineNotification
            v-if="alertsState.status === 'error'"
            kind="error"
            :title="$t('alerting.cannot_retrieve_alerts')"
            :description="alertsState.error?.message"
            class="mb-6"
          />

          <!-- loading skeleton -->
          <NeSkeleton v-else-if="alertsState.status === 'pending'" :lines="6" />

          <!-- empty with defaults -->
          <NeEmptyState
            v-else-if="!alertsState.data?.length && !hasActiveFilters"
            :title="$t('alerting.no_alerts')"
            :description="$t('alerting.no_alerts_description')"
            :icon="faBell"
          />

          <!-- empty with filters -->
          <NeEmptyState
            v-else-if="!alertsState.data?.length"
            :title="$t('alerting.no_alerts_found')"
            :description="$t('common.try_changing_search_filters')"
            :icon="faBell"
          />

          <!-- alerts list -->
          <div v-else class="space-y-3">
            <div
              v-for="alert in alertsState.data"
              :key="alert.fingerprint"
              class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-gray-700 dark:bg-gray-900"
            >
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div class="flex flex-wrap items-center gap-2">
                  <FontAwesomeIcon
                    :icon="faExclamationTriangle"
                    :class="[
                      'h-5 w-5 shrink-0',
                      alert.labels.severity === 'critical'
                        ? 'text-red-500'
                        : alert.labels.severity === 'warning'
                          ? 'text-amber-500'
                          : 'text-blue-400',
                    ]"
                    aria-hidden="true"
                  />
                  <span class="text-base font-semibold">{{ alert.labels.alertname || '-' }}</span>
                  <NeBadgeV2
                    v-if="alert.labels.severity"
                    :kind="getSeverityBadgeKind(alert.labels.severity)"
                    size="xs"
                  >
                    {{ alert.labels.severity }}
                  </NeBadgeV2>
                  <NeBadgeV2 :kind="getStateBadgeKind(alert.status?.state)" size="xs">
                    {{ alert.status?.state || '-' }}
                  </NeBadgeV2>
                </div>
                <div class="text-xs text-gray-400 dark:text-gray-500">
                  {{ $t('alerting.starts_at') }}:
                  {{
                    alert.startsAt ? formatDateTimeNoSeconds(new Date(alert.startsAt), locale) : '-'
                  }}
                </div>
              </div>

              <div
                v-if="getAlertSummaryText(alert) || getAlertDescriptionText(alert)"
                class="mt-2 space-y-1"
              >
                <div
                  v-if="getAlertSummaryText(alert)"
                  class="text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ getAlertSummaryText(alert) }}
                </div>
                <div
                  v-if="getAlertDescriptionText(alert)"
                  class="text-sm text-gray-600 dark:text-gray-400"
                >
                  {{ getAlertDescriptionText(alert) }}
                </div>
              </div>

              <div class="mt-3 flex flex-wrap gap-2">
                <template v-for="(val, key) in alert.labels" :key="key">
                  <span
                    v-if="key !== 'alertname' && key !== 'severity'"
                    class="rounded bg-gray-100 px-2 py-0.5 font-mono text-xs text-gray-600 dark:bg-gray-800 dark:text-gray-400"
                  >
                    {{ key }}={{ val }}
                  </span>
                </template>
              </div>
            </div>
          </div>
        </template>

        <!-- ══ CONFIGURATION TAB ══════════════════════════════════════════════ -->
        <template v-else-if="selectedPageTab === 'config'">
          <!-- errors -->
          <NeInlineNotification
            v-if="fetchError"
            kind="error"
            :title="$t('alerting.cannot_retrieve_config')"
            :description="fetchError"
            class="mb-6"
          />
          <NeInlineNotification
            v-if="saveError"
            kind="error"
            :title="$t('alerting.cannot_save_config')"
            :description="saveError"
            class="mb-6"
          />
          <NeInlineNotification
            v-if="disableError"
            kind="error"
            :title="$t('alerting.cannot_disable_alerts')"
            :description="disableError"
            class="mb-6"
          />

          <!-- loading skeleton -->
          <NeSkeleton v-if="isLoading && !config" :lines="10" />

          <template v-else-if="config || rawYaml">
            <!-- config sub-tabs -->
            <NeTabs
              :tabs="configTabs"
              :selected="selectedConfigTab"
              :sr-tabs-label="$t('ne_tabs.tabs')"
              :sr-select-tab-label="$t('ne_tabs.select_a_tab')"
              class="mb-6"
              @select-tab="selectedConfigTab = $event"
            />

            <!-- structured view -->
            <template v-if="selectedConfigTab === 'structured'">
              <!-- edit mode -->
              <template v-if="isEditMode">
                <NeCard class="mb-6">
                  <NeHeading tag="h6" class="mb-4">{{ $t('alerting.config_editor') }}</NeHeading>
                  <NeInlineNotification
                    v-if="editJsonError"
                    kind="error"
                    :title="editJsonError"
                    class="mb-4"
                  />
                  <NeTextArea
                    v-model="editJson"
                    :rows="20"
                    :label="$t('alerting.paste_json_config')"
                    class="font-mono text-xs"
                  />
                  <div class="mt-4 flex gap-3">
                    <NeButton kind="primary" :loading="isSaving" @click="saveConfig">
                      <template #prefix>
                        <FontAwesomeIcon :icon="faCheck" />
                      </template>
                      {{ $t('alerting.save_config') }}
                    </NeButton>
                    <NeButton kind="tertiary" :disabled="isSaving" @click="cancelEdit">
                      <template #prefix>
                        <FontAwesomeIcon :icon="faXmark" />
                      </template>
                      {{ $t('common.cancel') }}
                    </NeButton>
                  </div>
                </NeCard>
              </template>

              <!-- view mode -->
              <template v-else>
                <NeCard class="mb-6">
                  <div class="mb-4 flex items-center justify-between gap-4">
                    <div class="flex items-center gap-2">
                      <NeHeading tag="h6">{{ $t('alerting.config_json') }}</NeHeading>
                      <UpdatingSpinner v-if="isLoading" />
                    </div>
                    <NeButton kind="tertiary" size="sm" @click="startEdit">
                      <template #prefix>
                        <FontAwesomeIcon :icon="faPenToSquare" />
                      </template>
                      {{ $t('alerting.edit_config') }}
                    </NeButton>
                  </div>

                  <div v-if="config" class="divide-y divide-gray-200 dark:divide-gray-700">
                    <DataItem>
                      <template #label>{{ $t('alerting.mail_enabled') }}</template>
                      <template #data>
                        <NeBadgeV2 :kind="config.mail_enabled ? 'green' : 'gray'" size="xs">
                          {{ config.mail_enabled ? $t('common.enabled') : $t('common.disabled') }}
                        </NeBadgeV2>
                      </template>
                    </DataItem>
                    <DataItem>
                      <template #label>{{ $t('alerting.mail_addresses') }}</template>
                      <template #data>{{ formatMailAddresses(config.mail_addresses) }}</template>
                    </DataItem>
                    <DataItem>
                      <template #label>{{ $t('alerting.webhook_enabled') }}</template>
                      <template #data>
                        <NeBadgeV2 :kind="config.webhook_enabled ? 'green' : 'gray'" size="xs">
                          {{
                            config.webhook_enabled ? $t('common.enabled') : $t('common.disabled')
                          }}
                        </NeBadgeV2>
                      </template>
                    </DataItem>
                    <DataItem v-if="config.webhook_receivers?.length">
                      <template #label>{{ $t('alerting.webhook_receivers') }}</template>
                      <template #data>
                        <div class="space-y-1">
                          <div
                            v-for="recv in config.webhook_receivers"
                            :key="recv.name"
                            class="font-mono text-xs"
                          >
                            {{ recv.name }}: {{ recv.url }}
                          </div>
                        </div>
                      </template>
                    </DataItem>
                    <DataItem v-if="config.email_template_lang">
                      <template #label>{{ $t('alerting.email_template_lang') }}</template>
                      <template #data>{{ config.email_template_lang }}</template>
                    </DataItem>
                  </div>

                  <div v-if="config?.severities?.length" class="mt-6">
                    <NeHeading tag="h6" class="mb-3">
                      {{ $t('alerting.per_severity_overrides') }}
                    </NeHeading>
                    <div
                      v-for="sv in config.severities"
                      :key="sv.severity"
                      class="mb-3 rounded-lg border border-gray-200 p-3 dark:border-gray-700"
                    >
                      <div class="mb-2 flex items-center gap-2">
                        <NeBadgeV2
                          :kind="
                            sv.severity === 'critical'
                              ? 'rose'
                              : sv.severity === 'warning'
                                ? 'amber'
                                : 'blue'
                          "
                          size="xs"
                        >
                          {{ sv.severity }}
                        </NeBadgeV2>
                      </div>
                      <div class="grid grid-cols-2 gap-2 text-sm">
                        <div>{{ $t('alerting.mail_enabled') }}: {{ sv.mail_enabled ?? '-' }}</div>
                        <div>
                          {{ $t('alerting.webhook_enabled') }}: {{ sv.webhook_enabled ?? '-' }}
                        </div>
                        <div v-if="sv.mail_addresses?.length">
                          {{ $t('alerting.mail_addresses') }}: {{ sv.mail_addresses.join(', ') }}
                        </div>
                      </div>
                    </div>
                  </div>

                  <div v-if="config?.systems?.length" class="mt-6">
                    <NeHeading tag="h6" class="mb-3">
                      {{ $t('alerting.per_system_overrides') }}
                    </NeHeading>
                    <div
                      v-for="sys in config.systems"
                      :key="sys.system_key"
                      class="mb-3 rounded-lg border border-gray-200 p-3 dark:border-gray-700"
                    >
                      <div class="mb-2 font-mono text-xs text-gray-500">{{ sys.system_key }}</div>
                      <div class="grid grid-cols-2 gap-2 text-sm">
                        <div>{{ $t('alerting.mail_enabled') }}: {{ sys.mail_enabled ?? '-' }}</div>
                        <div>
                          {{ $t('alerting.webhook_enabled') }}: {{ sys.webhook_enabled ?? '-' }}
                        </div>
                        <div v-if="sys.mail_addresses?.length">
                          {{ $t('alerting.mail_addresses') }}: {{ sys.mail_addresses.join(', ') }}
                        </div>
                      </div>
                    </div>
                  </div>
                </NeCard>

                <!-- disable alerts section -->
                <NeCard>
                  <NeHeading tag="h6" class="mb-3 text-red-600 dark:text-red-400">
                    <FontAwesomeIcon :icon="faTrash" class="mr-2 h-4 w-4" />
                    {{ $t('alerting.disable_alerts') }}
                  </NeHeading>
                  <p class="mb-4 text-sm text-gray-500 dark:text-gray-400">
                    {{ $t('alerting.disable_alerts_confirmation') }}
                  </p>
                  <template v-if="!isConfirmDisable">
                    <NeButton kind="danger" size="sm" @click="isConfirmDisable = true">
                      <template #prefix>
                        <FontAwesomeIcon :icon="faTrash" />
                      </template>
                      {{ $t('alerting.disable_alerts') }}
                    </NeButton>
                  </template>
                  <template v-else>
                    <div class="flex items-center gap-3">
                      <NeButton kind="danger" :loading="isDisabling" @click="disableAlerts">
                        <template #prefix>
                          <FontAwesomeIcon :icon="faCheck" />
                        </template>
                        {{ $t('common.delete') }}
                      </NeButton>
                      <NeButton
                        kind="tertiary"
                        :disabled="isDisabling"
                        @click="isConfirmDisable = false"
                      >
                        {{ $t('common.cancel') }}
                      </NeButton>
                    </div>
                  </template>
                </NeCard>
              </template>
            </template>

            <!-- YAML view -->
            <template v-else-if="selectedConfigTab === 'yaml'">
              <NeCard>
                <div class="mb-4 flex items-center justify-between gap-4">
                  <NeHeading tag="h6">{{ $t('alerting.raw_yaml') }}</NeHeading>
                  <NeButton kind="tertiary" size="sm" @click="copyYaml">
                    <template #prefix>
                      <FontAwesomeIcon :icon="yamlCopied ? faCheck : faCopy" />
                    </template>
                    {{ yamlCopied ? $t('alerting.yaml_copied') : $t('alerting.copy_yaml') }}
                  </NeButton>
                </div>
                <pre
                  class="overflow-auto rounded-lg bg-gray-50 p-4 font-mono text-xs text-gray-800 dark:bg-gray-800 dark:text-gray-200"
                  >{{ rawYaml || $t('alerting.no_config') }}</pre
                >
              </NeCard>
            </template>
          </template>

          <!-- no config fallback -->
          <NeCard v-else-if="!isLoading">
            <div class="py-8 text-center text-gray-500 dark:text-gray-400">
              <FontAwesomeIcon :icon="faCog" class="mb-3 h-12 w-12 opacity-30" />
              <p class="text-lg font-semibold">{{ $t('alerting.no_config') }}</p>
              <p class="mt-1 text-sm">{{ $t('alerting.no_config_description') }}</p>
              <NeButton
                kind="primary"
                class="mt-4"
                @click="
                  () => {
                    isEditMode = true
                    selectedConfigTab = 'structured'
                  }
                "
              >
                <template #prefix>
                  <FontAwesomeIcon :icon="faPenToSquare" />
                </template>
                {{ $t('alerting.edit_config') }}
              </NeButton>
            </div>
          </NeCard>
        </template>
      </template>
    </template>
  </div>
</template>
