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
  focusElement,
} from '@nethesis/vue-components'
import { ref, useTemplateRef } from 'vue'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import {
  ALERTS_ALERTS_KEY,
  ALERTS_TOTALS_KEY,
  ALERT_ACTIVITY_KEY,
  createAlertAssignment,
  type Alert,
} from '@/lib/alerts'
import { useNotificationsStore } from '@/stores/notifications'
import AlertEventsTimeline from '@/components/alerts/AlertEventsTimeline.vue'

const { isShown = false, alert = undefined } = defineProps<{
  isShown: boolean
  alert: Alert | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const note = ref('')
const noteRef = useTemplateRef<HTMLTextAreaElement>('noteRef')

const {
  mutate: assignAlertMutate,
  isLoading: assignAlertLoading,
  reset: assignAlertReset,
  error: assignAlertError,
} = useMutation({
  mutation: async () => {
    if (!alert) return
    return createAlertAssignment(
      alert.fingerprint,
      alert.labels?.organization_id ?? '',
      note.value.trim() || undefined,
    )
  },
  onSuccess() {
    const alertname = alert?.labels?.alertname ?? ''
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('alerts.alert_assigned'),
        description: t('alerts.alert_assigned_description', { name: alertname }),
      })
    }, 500)
    closeDrawer()
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [ALERTS_ALERTS_KEY] })
    queryCache.invalidateQueries({ key: [ALERTS_TOTALS_KEY] })
    queryCache.invalidateQueries({ key: [ALERT_ACTIVITY_KEY] })
  },
})

function onShow() {
  assignAlertReset()
  note.value = ''
  focusElement(noteRef)
}

function closeDrawer() {
  emit('close')
}

function handleSubmit() {
  assignAlertMutate()
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="t('alerts.assign_alert_title', { alertname: alert?.labels?.alertname ?? '' })"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent="handleSubmit">
      <div class="space-y-6">
        <!-- Note (optional) -->
        <NeTextArea
          ref="noteRef"
          v-model="note"
          :label="t('alerts.assign_note')"
          :placeholder="t('alerts.assign_note_placeholder')"
          :optional="true"
          :optional-label="t('common.optional')"
          :disabled="assignAlertLoading"
        />
        <!-- Error notification -->
        <NeInlineNotification
          v-if="assignAlertError"
          kind="error"
          :title="t('alerts.cannot_assign_alert')"
          :description="(assignAlertError as Error).message"
        />
        <!-- Activity timeline -->
        <AlertEventsTimeline :alert="alert" />
      </div>
      <!-- footer -->
      <hr class="my-8" />
      <div class="flex justify-end">
        <NeButton
          kind="tertiary"
          size="lg"
          class="mr-3"
          :disabled="assignAlertLoading"
          @click.prevent="closeDrawer"
        >
          {{ $t('common.cancel') }}
        </NeButton>
        <NeButton
          type="submit"
          kind="primary"
          size="lg"
          :disabled="assignAlertLoading"
          :loading="assignAlertLoading"
          @click.prevent="handleSubmit"
        >
          {{ t('alerts.assign_to_me') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
