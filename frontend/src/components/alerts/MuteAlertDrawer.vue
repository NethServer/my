<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeInlineNotification,
  NeSideDrawer,
  NeTextArea,
  NeTextInput,
  focusElement,
} from '@nethesis/vue-components'
import { ref, useTemplateRef } from 'vue'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import {
  ALERTS_ALERTS_KEY,
  ALERTS_SILENCES_KEY,
  ALERTS_TOTALS_KEY,
  createSystemAlertSilence,
  type Alert,
} from '@/lib/alerts'
import { useNotificationsStore } from '@/stores/notifications'

const { isShown = false, alert = undefined } = defineProps<{
  isShown: boolean
  alert: Alert | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const endsAt = ref('')
const notes = ref('')
const endsAtRef = useTemplateRef<HTMLInputElement>('endsAtRef')
const endsAtError = ref('')

const {
  mutate: muteAlertMutate,
  isLoading: muteAlertLoading,
  reset: muteAlertReset,
  error: muteAlertError,
} = useMutation({
  mutation: async () => {
    if (!alert?.labels?.system_id) return
    return createSystemAlertSilence(
      alert.labels.system_id,
      alert.fingerprint,
      notes.value.trim() || undefined,
      endsAt.value ? new Date(endsAt.value).toISOString() : undefined,
    )
  },
  onSuccess() {
    const alertname = alert?.labels?.alertname ?? ''
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('alerts.alert_muted'),
        description: t('alerts.alert_muted_description', { name: alertname }),
      })
    }, 500)
    closeDrawer()
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [ALERTS_ALERTS_KEY] })
    queryCache.invalidateQueries({ key: [ALERTS_SILENCES_KEY] })
    queryCache.invalidateQueries({ key: [ALERTS_TOTALS_KEY] })
  },
})

function onShow() {
  clearErrors()
  endsAt.value = ''
  notes.value = ''
  focusElement(endsAtRef)
}

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  muteAlertReset()
  endsAtError.value = ''
}

function validate(): boolean {
  clearErrors()
  if (!endsAt.value) {
    endsAtError.value = t('alerts.mute_until_date_required')
    return false
  }
  if (new Date(endsAt.value) <= new Date()) {
    endsAtError.value = t('alerts.mute_until_date_future')
    return false
  }
  return true
}

function handleSubmit() {
  if (!validate()) return
  muteAlertMutate()
}

function getMinDateTime(): string {
  const d = new Date(Date.now() + 60_000)
  return d.toISOString().slice(0, 16)
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="t('alerts.mute_alert_title', { alertname: alert?.labels?.alertname ?? '' })"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent="handleSubmit">
      <div class="space-y-6">
        <!-- Mute until date/time -->
        <NeTextInput
          ref="endsAtRef"
          v-model="endsAt"
          type="datetime-local"
          :label="t('alerts.mute_until_date')"
          :min="getMinDateTime()"
          :invalid-message="endsAtError"
          :disabled="muteAlertLoading"
        />
        <!-- Notes (optional) -->
        <NeTextArea
          v-model="notes"
          :label="t('alerts.mute_notes')"
          :placeholder="t('alerts.mute_notes_placeholder')"
          :optional="true"
          :optional-label="t('common.optional')"
          :disabled="muteAlertLoading"
        />
        <!-- Error notification -->
        <NeInlineNotification
          v-if="muteAlertError"
          kind="error"
          :title="t('alerts.cannot_mute_alert')"
          :description="(muteAlertError as Error).message"
        />
      </div>
      <!-- footer -->
      <hr class="my-8" />
      <div class="flex justify-end">
        <NeButton
          kind="tertiary"
          size="lg"
          class="mr-3"
          :disabled="muteAlertLoading"
          @click.prevent="closeDrawer"
        >
          {{ $t('common.cancel') }}
        </NeButton>
        <NeButton
          type="submit"
          kind="primary"
          size="lg"
          :disabled="muteAlertLoading"
          :loading="muteAlertLoading"
          @click.prevent="handleSubmit"
        >
          {{ t('alerts.mute_alert') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
