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
import { faBellSlash, faTrash } from '@fortawesome/free-solid-svg-icons'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import { useLoginStore } from '@/stores/login'
import {
  deleteSystemAlertSilence,
  getSystemAlertSilences,
  type AlertmanagerSilence,
} from '@/lib/alerting'
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatDateTimeNoSeconds } from '@/lib/dateTime'
import { canManageSystems } from '@/lib/permissions'
import { useNotificationsStore } from '@/stores/notifications'
import UpdatingSpinner from '@/components/UpdatingSpinner.vue'

const { locale, t } = useI18n()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()
const { state: systemDetail } = useSystemDetail()

const silences = ref<AlertmanagerSilence[]>([])
const isLoading = ref(false)
const error = ref<string | null>(null)
const deletingId = ref<string | null>(null)
const loadedSystemId = ref('')
let requestId = 0

const systemId = computed(() => systemDetail.value.data?.id || '')

function getSilenceBadgeKind(state: string | undefined): NeBadgeV2Kind {
  switch (state?.toLowerCase()) {
    case 'active':
      return 'green'
    case 'pending':
      return 'amber'
    default:
      return 'gray'
  }
}

function getSilenceStateLabel(state: string | undefined) {
  switch (state?.toLowerCase()) {
    case 'active':
      return t('alerting.silence_status_active')
    case 'pending':
      return t('alerting.silence_status_pending')
    default:
      return state || '-'
  }
}

function getSilencedAlertName(silence: AlertmanagerSilence): string {
  return silence.matchers.find((m) => m.name === 'alertname')?.value || '-'
}

async function loadSilences(sysId: string) {
  const currentRequestId = ++requestId

  if (!sysId || !loginStore.jwtToken) {
    silences.value = []
    error.value = null
    isLoading.value = false
    loadedSystemId.value = ''
    return
  }

  isLoading.value = true
  error.value = null
  if (loadedSystemId.value !== sysId) {
    silences.value = []
  }

  try {
    const result = await getSystemAlertSilences(sysId)
    if (currentRequestId !== requestId) {
      return
    }
    silences.value = result
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

async function deleteSilence(silenceId: string) {
  const sysId = systemId.value
  if (!sysId) {
    return
  }

  deletingId.value = silenceId

  try {
    await deleteSystemAlertSilence(sysId, silenceId)
    notificationsStore.createNotification({
      kind: 'success',
      title: t('alerting.silence_deleted'),
      description: t('alerting.silence_deleted_description'),
    })
    void loadSilences(sysId)
  } catch (e: unknown) {
    notificationsStore.createNotification({
      kind: 'error',
      title: t('alerting.cannot_delete_silence'),
      description: e instanceof Error ? e.message : String(e),
    })
  } finally {
    deletingId.value = null
  }
}

watch(
  systemId,
  (sysId) => {
    void loadSilences(sysId)
  },
  { immediate: true },
)
</script>

<template>
  <NeCard class="col-span-full">
    <div class="mb-4 flex flex-col items-start justify-between gap-4 xl:flex-row">
      <div class="flex flex-wrap items-center gap-2">
        <FontAwesomeIcon :icon="faBellSlash" class="h-5 w-5 text-gray-500 dark:text-gray-400" />
        <NeHeading tag="h6">{{ $t('alerting.silences_card_title') }}</NeHeading>
        <NeBadgeV2 kind="gray" size="xs">{{ silences.length }}</NeBadgeV2>
      </div>
      <UpdatingSpinner v-if="isLoading && silences.length > 0" />
    </div>

    <NeInlineNotification
      v-if="error"
      kind="error"
      :title="$t('alerting.cannot_retrieve_silences')"
      :description="error"
      class="mb-4"
    />

    <NeSkeleton v-if="isLoading && !silences.length" :lines="4" />
    <NeEmptyState
      v-else-if="!silences.length && !error"
      :title="$t('alerting.no_silences')"
      :description="$t('alerting.no_silences_description')"
      :icon="faBellSlash"
    />
    <NeTable
      v-else-if="silences.length"
      :aria-label="$t('alerting.silences_card_title')"
      card-breakpoint="lg"
    >
      <NeTableHead>
        <NeTableHeadCell>{{ $t('alerting.alertname') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('common.status') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerting.silence_comment') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerting.silence_created_by') }}</NeTableHeadCell>
        <NeTableHeadCell>{{ $t('alerting.ends_at') }}</NeTableHeadCell>
        <NeTableHeadCell v-if="canManageSystems()">{{ $t('common.actions') }}</NeTableHeadCell>
      </NeTableHead>
      <NeTableBody>
        <NeTableRow v-for="silence in silences" :key="silence.id">
          <NeTableCell :data-label="$t('alerting.alertname')">
            <span class="font-medium">{{ getSilencedAlertName(silence) }}</span>
          </NeTableCell>
          <NeTableCell :data-label="$t('common.status')">
            <NeBadgeV2 :kind="getSilenceBadgeKind(silence.status?.state)" size="xs">
              {{ getSilenceStateLabel(silence.status?.state) }}
            </NeBadgeV2>
          </NeTableCell>
          <NeTableCell :data-label="$t('alerting.silence_comment')">
            {{ silence.comment || '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('alerting.silence_created_by')">
            {{ silence.createdBy || '-' }}
          </NeTableCell>
          <NeTableCell :data-label="$t('alerting.ends_at')">
            {{ silence.endsAt ? formatDateTimeNoSeconds(new Date(silence.endsAt), locale) : '-' }}
          </NeTableCell>
          <NeTableCell v-if="canManageSystems()" :data-label="$t('common.actions')">
            <div class="-ml-2.5 flex gap-2 xl:ml-0 xl:justify-end">
              <NeButton
                kind="tertiary"
                size="sm"
                :disabled="deletingId === silence.id"
                :loading="deletingId === silence.id"
                @click="deleteSilence(silence.id)"
              >
                <template #prefix>
                  <FontAwesomeIcon :icon="faTrash" aria-hidden="true" />
                </template>
                {{ $t('alerting.disable_silence') }}
              </NeButton>
            </div>
          </NeTableCell>
        </NeTableRow>
      </NeTableBody>
    </NeTable>
  </NeCard>
</template>
