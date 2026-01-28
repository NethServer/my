<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeSideDrawer,
  NeTextInput,
  NeInlineNotification,
  NeTextArea,
  focusElement,
} from '@nethesis/vue-components'
import { ref, useTemplateRef } from 'vue'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { getValidationIssues, isValidationError } from '@/lib/validation'
import {
  APPLICATIONS_KEY,
  getDisplayName,
  type Application,
  putApplication,
} from '@/lib/applications/applications'
import type { AxiosError } from 'axios'

const { isShown = false, currentApplication = undefined } = defineProps<{
  isShown: boolean
  currentApplication: Application | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const notes = ref('')
const notesRef = useTemplateRef<HTMLTextAreaElement>('notesRef')

const {
  mutate: setNotesMutate,
  isLoading: setNotesLoading,
  reset: setNotesReset,
  error: setNotesError,
} = useMutation({
  mutation: async (application: Application) => {
    return putApplication(application)
  },
  onSuccess(data, vars) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('applications.set_application_notes'),
        description: t('applications.set_application_notes_description', {
          application: getDisplayName(vars),
        }),
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error setting application notes:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'applications')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [APPLICATIONS_KEY] })
  },
})

const validationIssues = ref<Record<string, string[]>>({})

function onShow() {
  clearErrors()
  notes.value = currentApplication?.notes || ''
  focusElement(notesRef)
}

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  setNotesReset()
  validationIssues.value = {}
}

async function saveApplication() {
  clearErrors()

  if (!currentApplication) {
    return
  }

  const application: Application = {
    ...currentApplication,
    notes: notes.value,
  }
  setNotesMutate(application)
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="
      currentApplication?.notes ? $t('applications.edit_notes') : $t('applications.add_notes')
    "
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <!-- name -->
        <NeTextInput
          :value="currentApplication ? getDisplayName(currentApplication) : ''"
          :label="$t('applications.application')"
          readonly
        />
        <!-- notes -->
        <NeTextArea
          ref="notesRef"
          v-model="notes"
          @blur="notes = notes.trim()"
          :label="$t('applications.notes')"
          :disabled="setNotesLoading"
          :invalid-message="validationIssues.notes?.[0] ? $t(validationIssues.notes[0]) : ''"
          :optional="true"
          :optional-label="t('common.optional')"
        />
        <!-- set notes error notification -->
        <NeInlineNotification
          v-if="setNotesError?.message && !isValidationError(setNotesError)"
          kind="error"
          :title="t('applications.cannot_set_notes')"
          :description="setNotesError.message"
        />
      </div>
      <!-- footer -->
      <hr class="my-8" />
      <div class="flex justify-end">
        <NeButton kind="tertiary" size="lg" class="mr-3" @click.prevent="closeDrawer">
          {{ $t('common.cancel') }}
        </NeButton>
        <NeButton
          type="submit"
          kind="primary"
          size="lg"
          :disabled="setNotesLoading"
          :loading="setNotesLoading"
          @click.prevent="saveApplication"
        >
          {{ $t('common.save') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
