<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeInlineNotification, NeSideDrawer, NeTextArea } from '@nethesis/vue-components'
import { VueDatePicker } from '@vuepic/vue-datepicker'
import { ref } from 'vue'
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
import { useThemeStore } from '@/stores/theme'

const { isShown = false, alert = undefined } = defineProps<{
  isShown: boolean
  alert: Alert | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()
const themeStore = useThemeStore()

const endsAt = ref<Date | null>(null)
const notes = ref('')
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
      endsAt.value ? endsAt.value.toISOString() : undefined,
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
  endsAt.value = null
  notes.value = ''
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
  if (endsAt.value <= new Date()) {
    endsAtError.value = t('alerts.mute_until_date_future')
    return false
  }
  return true
}

function handleSubmit() {
  if (!validate()) return
  muteAlertMutate()
}

function getMinDate(): Date {
  return new Date(Date.now() + 60_000)
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
        <div class="space-y-1.5">
          <label
            for="muteUntilDate"
            class="block text-sm font-medium text-gray-700 dark:text-gray-200"
          >
            {{ t('alerts.mute_until_date') }}
          </label>
          <VueDatePicker
            v-model="endsAt"
            class="vue-datepicker"
            :dark="!themeStore.isLight"
            :enable-time-picker="true"
            :enable-seconds="false"
            :min-date="getMinDate()"
            :disabled="muteAlertLoading"
            :placeholder="t('alerts.mute_until_date_placeholder')"
            :time-config="{ timePickerInline: true }"
            :input-attrs="{ id: 'muteUntilDate' }"
            auto-apply
            @update:model-value="endsAtError = ''"
          />
          <p v-if="endsAtError" class="text-sm text-rose-700 dark:text-rose-400">
            {{ endsAtError }}
          </p>
        </div>
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
