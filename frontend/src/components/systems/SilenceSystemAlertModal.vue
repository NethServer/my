<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification, NeModal, NeTextArea, NeTextInput } from '@nethesis/vue-components'
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
const endAt = ref('')

const alertName = computed(() => {
  if (!alert) {
    return '-'
  }
  return alert.labels.alertname || getAlertSummary(alert, locale.value) || alert.fingerprint
})

function defaultEndAt() {
  const d = new Date(Date.now() + 4 * 60 * 60 * 1000)
  // datetime-local requires "YYYY-MM-DDTHH:MM"
  return d.toISOString().slice(0, 16)
}

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
    endAt,
  }: {
    systemId: string
    fingerprint: string
    comment?: string
    endAt?: string
  }) => {
    return createSystemAlertSilence(systemId, fingerprint, comment, endAt)
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
  endAt.value = defaultEndAt()
  createSilenceReset()
}

function onPrimaryClick() {
  if (!alert || !systemId) {
    return
  }

  // Convert the datetime-local value to a full RFC3339 string
  const endAtRfc3339 = endAt.value ? new Date(endAt.value).toISOString() : undefined

  createSilenceMutate({
    systemId,
    fingerprint: alert.fingerprint,
    comment: comment.value,
    endAt: endAtRfc3339,
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
      <NeTextInput
        v-model="endAt"
        type="datetime-local"
        :label="$t('alerting.silence_end_at')"
        :helper-text="$t('alerting.silence_end_at_helper')"
      />
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
