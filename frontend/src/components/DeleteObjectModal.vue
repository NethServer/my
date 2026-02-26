<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { focusElement, NeInlineNotification, NeTextInput } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { ref, useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'

const {
  visible = false,
  title = '',
  primaryLabel = '',
  deleting = false,
  confirmationMessage = '',
  confirmationInput = '',
  errorTitle = '',
  errorDescription = '',
} = defineProps<{
  visible: boolean
  title: string
  primaryLabel: string
  deleting: boolean
  confirmationMessage?: string
  errorTitle?: string
  errorDescription?: string
  confirmationInput?: string
}>()

const emit = defineEmits(['show', 'close', 'primary-click'])

const { t } = useI18n()
const confirmationText = ref('')
const confirmationTextRef = useTemplateRef<HTMLTextAreaElement>('confirmationTextRef')
const invalidConfirmationText = ref('')

function onShow() {
  if (confirmationInput) {
    confirmationText.value = ''
    invalidConfirmationText.value = ''

    // focus confirmation text input when modal is shown
    focusElement(confirmationTextRef)
  }
  emit('show')
}

function onPrimaryClick() {
  invalidConfirmationText.value = ''

  if (confirmationInput && confirmationText.value.trim() !== confirmationInput.trim()) {
    // show error if confirmation text does not match
    invalidConfirmationText.value = t('delete_object_modal.confirmation_text_does_not_match', {
      confirmationText: confirmationInput,
    })
    focusElement(confirmationTextRef)
  } else {
    emit('primary-click')
  }
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="title"
    kind="warning"
    :primary-label="primaryLabel"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="danger"
    :primary-button-disabled="deleting"
    :primary-button-loading="deleting"
    :close-aria-label="$t('common.close')"
    @show="onShow"
    @close="emit('close')"
    @primary-click="onPrimaryClick"
  >
    <template v-if="confirmationMessage">
      <p>{{ confirmationMessage }}</p>
    </template>
    <!-- custom content if no confirmationMessage is provided -->
    <slot v-else />
    <!-- type confirmation text -->
    <NeTextInput
      v-if="confirmationInput"
      ref="confirmationTextRef"
      v-model="confirmationText"
      :label="
        t('delete_object_modal.type_to_confirm', {
          confirmationText: confirmationInput,
        })
      "
      :invalid-message="invalidConfirmationText"
      class="mt-4"
    />
    <NeInlineNotification
      v-if="errorTitle && errorDescription"
      kind="error"
      :title="errorTitle"
      :description="errorDescription"
      class="mt-4"
    />
  </NeModal>
</template>
