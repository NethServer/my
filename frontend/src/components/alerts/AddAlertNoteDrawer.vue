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
import { ref, watch, useTemplateRef } from 'vue'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'
import {
  ALERTS_ALERTS_KEY,
  ALERTS_TOTALS_KEY,
  ALERT_ACTIVITY_KEY,
  createAlertNote,
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

const text = ref('')
const textError = ref('')
const textRef = useTemplateRef<HTMLTextAreaElement>('textRef')

// Clear the validation error as soon as the user edits the field.
watch(text, () => {
  if (textError.value) textError.value = ''
})

const {
  mutate: addNoteMutate,
  isLoading: addNoteLoading,
  reset: addNoteReset,
  error: addNoteError,
} = useMutation({
  mutation: async () => {
    if (!alert) return
    return createAlertNote(
      alert.fingerprint,
      alert.labels?.organization_id ?? '',
      text.value.trim(),
    )
  },
  onSuccess() {
    const alertname = alert?.labels?.alertname ?? ''
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('alerts.comment_added'),
        description: t('alerts.comment_added_description', { name: alertname }),
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

function clearErrors() {
  addNoteReset()
  textError.value = ''
}

function onShow() {
  clearErrors()
  text.value = ''
  focusElement(textRef)
}

function closeDrawer() {
  emit('close')
}

function validate(): boolean {
  clearErrors()
  if (!text.value.trim()) {
    textError.value = t('alerts.comment_required')
    return false
  }
  return true
}

function handleSubmit() {
  if (!validate()) return
  addNoteMutate()
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="t('alerts.add_comment_title', { alertname: alert?.labels?.alertname ?? '' })"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent="handleSubmit">
      <div class="space-y-6">
        <!-- Comment text (required) -->
        <NeTextArea
          ref="textRef"
          v-model="text"
          :label="t('alerts.comment_text')"
          :placeholder="t('alerts.comment_text_placeholder')"
          :invalid-message="textError"
          :disabled="addNoteLoading"
        />
        <!-- Error notification -->
        <NeInlineNotification
          v-if="addNoteError"
          kind="error"
          :title="t('alerts.cannot_add_comment')"
          :description="(addNoteError as Error).message"
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
          :disabled="addNoteLoading"
          @click.prevent="closeDrawer"
        >
          {{ $t('common.cancel') }}
        </NeButton>
        <NeButton
          type="submit"
          kind="primary"
          size="lg"
          :disabled="addNoteLoading"
          :loading="addNoteLoading"
          @click.prevent="handleSubmit"
        >
          {{ t('alerts.add_comment') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
