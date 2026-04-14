<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification, NeModal, NeTextArea } from '@nethesis/vue-components'
import { useMutation } from '@pinia/colada'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { createSystemAlertSilence, getAlertSummary, type Alert } from '@/lib/alerting'
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
const comment = ref('')

const alertName = computed(() => {
  if (!alert) {
    return '-'
  }
  return alert.labels.alertname || getAlertSummary(alert, locale.value) || alert.fingerprint
})

const {
  mutate: createSilenceMutate,
  isLoading: createSilenceLoading,
  reset: createSilenceReset,
  error: createSilenceError,
} = useMutation({
  mutation: ({
    systemId,
    fingerprint,
    comment,
  }: {
    systemId: string
    fingerprint: string
    comment?: string
  }) => {
    return createSystemAlertSilence(systemId, fingerprint, comment)
  },
  onSuccess() {
    const silencedAlertName = alertName.value

    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('alerting.alert_silenced'),
        description: t('alerting.alert_silenced_description', {
          name: silencedAlertName,
        }),
      })
    }, 500)

    emit('success')
  },
  onError: (error) => {
    console.error('Error silencing alert:', error)
  },
})

function onShow() {
  comment.value = ''
  createSilenceReset()
}

function onPrimaryClick() {
  if (!alert || !systemId) {
    return
  }

  createSilenceMutate({
    systemId,
    fingerprint: alert.fingerprint,
    comment: comment.value,
  })
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('alerting.silence_alert')"
    kind="warning"
    :primary-label="$t('alerting.silence_alert')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="createSilenceLoading || !alert"
    :primary-button-loading="createSilenceLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="onPrimaryClick"
    @show="onShow"
  >
    <div class="space-y-4">
      <p>
        {{ t('alerting.silence_alert_confirmation', { name: alertName }) }}
      </p>
      <p class="text-sm text-gray-500 dark:text-gray-400">
        {{ $t('alerting.silence_alert_duration_notice') }}
      </p>
      <NeTextArea
        v-model="comment"
        :label="$t('alerting.silence_comment')"
        :helper-text="$t('alerting.silence_comment_helper')"
        :optional="true"
        :optional-label="$t('common.optional')"
      />
      <NeInlineNotification
        v-if="createSilenceError?.message"
        kind="error"
        :title="$t('alerting.cannot_silence_alert')"
        :description="createSilenceError.message"
      />
    </div>
  </NeModal>
</template>
