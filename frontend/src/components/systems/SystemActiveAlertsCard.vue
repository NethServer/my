<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeBadgeV2,
  NeCard,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
  NeEmptyState,
  type NeBadgeV2Kind,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faBell, faExclamationTriangle } from '@fortawesome/free-solid-svg-icons'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useLoginStore } from '@/stores/login'
import { getSystemActiveAlerts, type Alert } from '@/lib/alerting'
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'

const { locale } = useI18n()
const loginStore = useLoginStore()
const { state: systemDetail } = useSystemDetail()

const alerts = ref<Alert[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)

const organizationId = computed(() => systemDetail.value.data?.organization?.logto_id || '')
const systemKey = computed(() => systemDetail.value.data?.system_key || '')

watch(
  [organizationId, systemKey],
  async ([orgId, sysKey]) => {
    if (!orgId || !sysKey || !loginStore.jwtToken) return
    isLoading.value = true
    error.value = null
    try {
      alerts.value = await getSystemActiveAlerts(orgId, sysKey)
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      isLoading.value = false
    }
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
        <div v-if="alert.annotations?.summary" class="text-sm text-gray-600 dark:text-gray-400">
          {{ alert.annotations.summary }}
        </div>
        <div class="text-xs text-gray-400 dark:text-gray-500">
          {{ $t('alerting.starts_at') }}:
          {{ alert.startsAt ? formatDateTimeNoSeconds(new Date(alert.startsAt), locale) : '-' }}
        </div>
      </div>
    </div>
  </NeCard>
</template>
