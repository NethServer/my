<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeBadgeV2,
  NeButton,
  NeCard,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
  NeEmptyState,
  type NeBadgeV2Kind,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faBell, faBellSlash, faExclamationTriangle } from '@fortawesome/free-solid-svg-icons'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useLoginStore } from '@/stores/login'
import {
  getAlertDescription,
  getAlertSummary,
  getSystemActiveAlerts,
  type Alert,
} from '@/lib/alerting'
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import { canManageSystems } from '@/lib/permissions'
import SilenceSystemAlertModal from './SilenceSystemAlertModal.vue'

const { locale } = useI18n()
const loginStore = useLoginStore()
const { state: systemDetail } = useSystemDetail()

const alerts = ref<Alert[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const currentAlert = ref<Alert | undefined>()
const isShownSilenceAlertModal = ref(false)

const systemId = computed(() => systemDetail.value.data?.id || '')
const activeAlertsCount = computed(() => alerts.value.length)
let requestId = 0

function getAlertSummaryText(alert: Alert) {
  return getAlertSummary(alert, locale.value)
}

function getAlertDescriptionText(alert: Alert) {
  const description = getAlertDescription(alert, locale.value)
  return description !== getAlertSummaryText(alert) ? description : ''
}

function isAlertSilenced(alert: Alert) {
  return alert.status?.silencedBy?.length > 0
}

async function loadAlerts(sysId: string) {
  const currentRequestId = ++requestId

  if (!sysId || !loginStore.jwtToken) {
    alerts.value = []
    error.value = null
    isLoading.value = false
    return
  }

  isLoading.value = true
  error.value = null
  alerts.value = []

  try {
    const systemAlerts = await getSystemActiveAlerts(sysId)
    if (currentRequestId !== requestId) {
      return
    }

    alerts.value = systemAlerts
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
  isShownSilenceAlertModal.value = true
}

function closeSilenceAlertModal() {
  isShownSilenceAlertModal.value = false
  currentAlert.value = undefined
}

function onAlertSilenced() {
  closeSilenceAlertModal()
  if (systemId.value) {
    void loadAlerts(systemId.value)
  }
}

watch(
  systemId,
  (sysId) => {
    void loadAlerts(sysId)
  },
  { immediate: true },
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
    default:
      return 'gray'
  }
}
</script>

<template>
  <NeCard class="col-span-full">
    <div class="mb-4 flex items-center gap-2">
      <FontAwesomeIcon :icon="faBell" class="h-5 w-5 text-gray-500 dark:text-gray-400" />
      <NeHeading tag="h6">{{ $t('alerting.active_alerts_card_title') }}</NeHeading>
      <NeBadgeV2 :kind="activeAlertsCount ? 'rose' : 'gray'" size="xs">
        {{ activeAlertsCount }}
      </NeBadgeV2>
      <NeBadgeV2 kind="gray" size="xs" class="ml-1">ALPHA</NeBadgeV2>
    </div>

    <!-- error -->
    <NeInlineNotification
      v-if="error"
      kind="error"
      :title="$t('alerting.cannot_retrieve_system_alerts')"
      :description="error"
      class="mb-4"
    />

    <!-- loading -->
    <NeSkeleton v-else-if="isLoading" :lines="3" />

    <!-- empty state -->
    <NeEmptyState
      v-else-if="!alerts.length"
      :title="$t('alerting.no_active_alerts')"
      :description="$t('alerting.no_active_alerts_description')"
      :icon="faBell"
    />

    <!-- alerts list -->
    <div v-else class="space-y-3">
      <div
        v-for="alert in alerts"
        :key="alert.fingerprint"
        class="flex flex-col gap-1 rounded-lg border border-gray-200 p-3 dark:border-gray-700"
      >
        <div class="flex flex-wrap items-center gap-2">
          <FontAwesomeIcon
            :icon="faExclamationTriangle"
            class="h-4 w-4 shrink-0 text-amber-500"
            aria-hidden="true"
          />
          <span class="font-semibold">{{ alert.labels.alertname || '-' }}</span>
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
        <div v-if="getAlertSummaryText(alert) || getAlertDescriptionText(alert)" class="space-y-1">
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
        <div class="text-xs text-gray-400 dark:text-gray-500">
          {{ $t('alerting.starts_at') }}:
          {{ alert.startsAt ? formatDateTimeNoSeconds(new Date(alert.startsAt), locale) : '-' }}
        </div>
        <div v-if="canManageSystems() && !isAlertSilenced(alert)" class="mt-2">
          <NeButton kind="tertiary" size="sm" @click="showSilenceAlertModal(alert)">
            <template #prefix>
              <FontAwesomeIcon :icon="faBellSlash" aria-hidden="true" />
            </template>
            {{ $t('alerting.silence_alert') }}
          </NeButton>
        </div>
      </div>
    </div>
    <SilenceSystemAlertModal
      :visible="isShownSilenceAlertModal"
      :alert="currentAlert"
      :system-id="systemId"
      @close="closeSilenceAlertModal"
      @success="onAlertSilenced"
    />
  </NeCard>
</template>
