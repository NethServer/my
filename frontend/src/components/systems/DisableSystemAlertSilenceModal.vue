<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification, NeModal } from '@nethesis/vue-components'
import { useMutation } from '@pinia/colada'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  deleteSystemAlertSilence,
  getAlertSilenceIds,
  getAlertSummary,
  type Alert,
} from '@/lib/alerting'
import { useNotificationsStore } from '@/stores/notifications'

const {
  visible = false,
  alert = undefined,
  systemId = '',
} = defineProps<{
  visible: boolean
  alert: Alert | undefined
  systemId: string
}>()

const emit = defineEmits(['close', 'success'])

const { t, locale } = useI18n()
const notificationsStore = useNotificationsStore()

const alertName = computed(() => {
  if (!alert) {
    return '-'
  }
  return alert.labels.alertname || getAlertSummary(alert, locale.value) || alert.fingerprint
})

const silenceIds = computed(() => {
  return alert ? getAlertSilenceIds(alert) : []
})

const silenceCountText = computed(() => {
  return t('alerting.silences_count', { num: silenceIds.value.length }, silenceIds.value.length)
})

const {
  mutate: deleteSilenceMutate,
  isLoading: deleteSilenceLoading,
  reset: deleteSilenceReset,
  error: deleteSilenceError,
} = useMutation({
  mutation: async ({ systemId, silenceIds }: { systemId: string; silenceIds: string[] }) => {
    for (const silenceId of silenceIds) {
      await deleteSystemAlertSilence(systemId, silenceId)
    }
  },
  onSuccess() {
    const updatedAlertName = alertName.value

    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('alerting.silence_disabled'),
        description: t('alerting.silence_disabled_description', {
          name: updatedAlertName,
        }),
      })
    }, 500)

    emit('success')
  },
  onError: (error) => {
    console.error('Error disabling alert silence:', error)
  },
})

function onShow() {
  deleteSilenceReset()
}

function onPrimaryClick() {
  if (!alert || !systemId || silenceIds.value.length === 0) {
    return
  }

  deleteSilenceMutate({
    systemId,
    silenceIds: silenceIds.value,
  })
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('alerting.disable_silence')"
    kind="warning"
    :primary-label="$t('alerting.disable_silence')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="deleteSilenceLoading || silenceIds.length === 0"
    :primary-button-loading="deleteSilenceLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="onPrimaryClick"
    @show="onShow"
  >
    <div class="space-y-4">
      <p>
        {{ t('alerting.disable_silence_confirmation', { name: alertName }) }}
      </p>
      <p class="text-sm text-gray-500 dark:text-gray-400">
        {{ $t('alerting.disable_silence_notice', { count: silenceCountText }) }}
      </p>
      <NeInlineNotification
        v-if="deleteSilenceError?.message"
        kind="error"
        :title="$t('alerting.cannot_disable_silence')"
        :description="deleteSilenceError.message"
      />
    </div>
  </NeModal>
</template>
