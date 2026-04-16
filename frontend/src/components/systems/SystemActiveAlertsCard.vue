<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeBadgeV2,
  NeButton,
  NeCard,
  NeEmptyState,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
  NeTable,
  NeTableBody,
  NeTableCell,
  NeTableHead,
  NeTableHeadCell,
  NeTableRow,
  type NeBadgeV2Kind,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faBell, faBellSlash, faExclamationTriangle } from '@fortawesome/free-solid-svg-icons'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useLoginStore } from '@/stores/login'
import {
  getAlertDescription,
  getAlertSilenceIds,
  getAlertSummary,
  getSystemActiveAlerts,
  isAlertSilenced,
  type Alert,
} from '@/lib/alerting'
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import { canManageSystems } from '@/lib/permissions'
import SilenceSystemAlertModal from './SilenceSystemAlertModal.vue'
import DisableSystemAlertSilenceModal from './DisableSystemAlertSilenceModal.vue'
import UpdatingSpinner from '@/components/UpdatingSpinner.vue'

const { refreshNonce = 0, refreshPending = false } = defineProps<{
  refreshNonce?: number
  refreshPending?: boolean
}>()
const emit = defineEmits(['alerting-action-success'])
const { locale, t } = useI18n()
const loginStore = useLoginStore()
const { state: systemDetail } = useSystemDetail()

const alerts = ref<Alert[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const currentAlert = ref<Alert | undefined>()
const isShownSilenceAlertModal = ref(false)
const isShownDisableSilenceModal = ref(false)
const loadedSystemId = ref('')

const systemId = computed(() => systemDetail.value.data?.id || '')
const activeAlertsCount = computed(() => alerts.value.length)
const silencedAlertsCount = computed(() => alerts.value.filter(isAlertSilenced).length)
const sortedAlerts = computed(() => {
  const stateOrder: Record<string, number> = {
    active: 0,
    suppressed: 1,
    unprocessed: 2,
  }

  return [...alerts.value].sort((firstAlert, secondAlert) => {
    const firstStateOrder = stateOrder[firstAlert.status?.state?.toLowerCase() || ''] ?? 99
    const secondStateOrder = stateOrder[secondAlert.status?.state?.toLowerCase() || ''] ?? 99

    if (firstStateOrder !== secondStateOrder) {
      return firstStateOrder - secondStateOrder
    }

    const firstStartsAt = firstAlert.startsAt ? Date.parse(firstAlert.startsAt) : 0
    const secondStartsAt = secondAlert.startsAt ? Date.parse(secondAlert.startsAt) : 0
    if (firstStartsAt !== secondStartsAt) {
      return secondStartsAt - firstStartsAt
    }

    return (firstAlert.labels.alertname || '').localeCompare(secondAlert.labels.alertname || '')
  })
})
let requestId = 0

function getAlertSummaryText(alert: Alert) {
  return getAlertSummary(alert, locale.value)
}

function getAlertDescriptionText(alert: Alert) {
  const description = getAlertDescription(alert, locale.value)
  return description !== getAlertSummaryText(alert) ? description : ''
}

async function loadAlerts(sysId: string, options: { reset?: boolean } = {}) {
  const currentRequestId = ++requestId

  if (!sysId || !loginStore.jwtToken) {
    alerts.value = []
    error.value = null
    isLoading.value = false
    loadedSystemId.value = ''
    return
  }

  isLoading.value = true
  error.value = null
  if (options.reset || loadedSystemId.value !== sysId) {
    alerts.value = []
  }

  try {
    const systemAlerts = await getSystemActiveAlerts(sysId)
    if (currentRequestId !== requestId) {
      return
    }

    alerts.value = systemAlerts
    loadedSystemId.value = sysId
  } catch (e: unknown) {
    if (currentRequestId !== requestId) {
      return
    }

    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    if (currentRequestId === requestId) {
      isLoading.value = false
    }
  }
}

function showSilenceAlertModal(alert: Alert) {
  currentAlert.value = alert
  isShownDisableSilenceModal.value = false
  isShownSilenceAlertModal.value = true
}

function closeAlertActionModals() {
  isShownSilenceAlertModal.value = false
  isShownDisableSilenceModal.value = false
  currentAlert.value = undefined
}

function onAlertActionSuccess() {
  closeAlertActionModals()
  emit('alerting-action-success')
}

watch(
  systemId,
  (sysId) => {
    void loadAlerts(sysId)
  },
  { immediate: true },
)

watch(
  () => refreshPending,
  (pending) => {
    if (!pending || !systemId.value) {
      return
    }

    requestId += 1
    isLoading.value = true
    error.value = null
    alerts.value = []
  },
)

watch(
  () => refreshNonce,
  () => {
    if (!systemId.value) {
      return
    }

    void loadAlerts(systemId.value, { reset: true })
  },
)

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

function getStateBadgeKind(state: string | undefined): NeBadgeV2Kind {
  switch (state?.toLowerCase()) {
    case 'active':
      return 'rose'
    case 'suppressed':
      return 'gray'
    case 'unprocessed':
      return 'amber'
    default:
      return 'gray'
  }
}

function getAlertStateLabel(state: string | undefined) {
  switch (state?.toLowerCase()) {
    case 'active':
      return t('alerting.state_active')
    case 'suppressed':
      return t('alerting.state_suppressed')
    case 'unprocessed':
      return t('alerting.state_unprocessed')
    default:
      return state || '-'
  }
}

function getAlertSilenceCountText(alert: Alert) {
  const silenceIds = getAlertSilenceIds(alert)
  if (!silenceIds.length) {
    return '-'
  }

  return t('alerting.silences_count', { num: silenceIds.length }, silenceIds.length)
}

function isSuppressed(alert: Alert) {
  return alert.status?.state?.toLowerCase() === 'suppressed'
}
</script>

<template>
  <NeCard class="col-span-full">
    <div class="mb-4 flex flex-col items-start justify-between gap-4 xl:flex-row">
      <div class="flex flex-wrap items-center gap-2">
        <FontAwesomeIcon :icon="faBell" class="h-5 w-5 text-gray-500 dark:text-gray-400" />
        <NeHeading tag="h6">{{ $t('alerting.active_alerts_card_title') }}</NeHeading>
        <NeBadgeV2 :kind="activeAlertsCount ? 'rose' : 'gray'" size="xs">
          {{ activeAlertsCount }}
        </NeBadgeV2>
        <NeBadgeV2 v-if="silencedAlertsCount" kind="gray" size="xs">
          {{
            $t('alerting.silenced_alerts_count', { num: silencedAlertsCount }, silencedAlertsCount)
          }}
        </NeBadgeV2>
        <NeBadgeV2 kind="gray" size="xs" class="ml-1">ALPHA</NeBadgeV2>
      </div>
      <UpdatingSpinner v-if="isLoading && alerts.length > 0" />
    </div>

    <NeInlineNotification
      v-if="error"
      kind="error"
      :title="$t('alerting.cannot_retrieve_system_alerts')"
      :description="error"
      class="mb-4"
    />

    <NeSkeleton v-if="isLoading && !alerts.length" :lines="6" />
    <NeEmptyState
      v-else-if="!alerts.length && !error"
      :title="$t('alerting.no_active_alerts')"
      :description="$t('alerting.no_active_alerts_description')"
      :icon="faBell"
    />
    <NeTable
      v-else-if="alerts.length"
      :aria-label="$t('alerting.active_alerts_card_title')"
      card-breakpoint="lg"
    >
      <NeTableHead>
        <NeTableHeadCell>{{ $t('alerting.alertname') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerting.severity') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('common.status') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerting.summary') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerting.starts_at') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerting.silences') }}</NeTableHeadCell>
        <NeTableHeadCell v-if="canManageSystems()">{{ $t('common.actions') }}</NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow
          v-for="alert in sortedAlerts"
          :key="alert.fingerprint"
          :class="isSuppressed(alert) ? 'opacity-50' : ''"
        >
          <NeTableCell :data-label="$t('alerting.alertname')">
            <div class="flex items-center gap-2">
              <FontAwesomeIcon
                :icon="faExclamationTriangle"
                class="h-4 w-4 shrink-0 text-amber-500"
                aria-hidden="true"
              />
              <span class="font-medium">{{ alert.labels.alertname || '-' }}</span>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('alerting.severity')">
            <NeBadgeV2
              v-if="alert.labels.severity"
              :kind="getSeverityBadgeKind(alert.labels.severity)"
              size="xs"
            >
              {{ alert.labels.severity }}
            </NeBadgeV2>
            <span v-else>-</span>
          </NeTableCell>
          <NeTableCell :data-label="$t('common.status')">
            <NeBadgeV2 :kind="getStateBadgeKind(alert.status?.state)" size="xs">
              {{ getAlertStateLabel(alert.status?.state) }}
            </NeBadgeV2>
          </NeTableCell>
          <NeTableCell :data-label="$t('alerting.summary')">
            <div class="space-y-1">
              <div class="font-medium text-gray-700 dark:text-gray-300">
                {{ getAlertSummaryText(alert) || '-' }}
              </div>
              <div
                v-if="getAlertDescriptionText(alert)"
                class="text-sm text-gray-600 dark:text-gray-400"
              >
                {{ getAlertDescriptionText(alert) }}
              </div>
            </div>
          </NeTableCell>
          <NeTableCell :data-label="$t('alerting.starts_at')">
            {{ alert.startsAt ? formatDateTimeNoSeconds(new Date(alert.startsAt), locale) : '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('alerting.silences')">
            <NeBadgeV2 v-if="isAlertSilenced(alert)" kind="gray" size="xs">
              {{ getAlertSilenceCountText(alert) }}
            </NeBadgeV2>
            <span v-else>-</span>
          </NeTableCell>
          <NeTableCell v-if="canManageSystems()" :data-label="$t('common.actions')">
            <div v-if="!isSuppressed(alert)" class="-ml-2.5 flex gap-2 xl:ml-0 xl:justify-end">
              <NeButton kind="tertiary" size="sm" @click="showSilenceAlertModal(alert)">
                <template #prefix>
                  <FontAwesomeIcon :icon="faBellSlash" aria-hidden="true" />
                </template>
                {{ $t('alerting.silence_alert') }}
              </NeButton>
            </div>
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
    </NeTable>
    <SilenceSystemAlertModal
      :visible="isShownSilenceAlertModal"
      :alert="currentAlert"
      :system-id="systemId"
      @close="closeAlertActionModals"
      @success="onAlertActionSuccess"
    />
    <DisableSystemAlertSilenceModal
      :visible="isShownDisableSilenceModal"
      :alert="currentAlert"
      :system-id="systemId"
      @close="closeAlertActionModals"
      @success="onAlertActionSuccess"
    />
  </NeCard>
</template>
