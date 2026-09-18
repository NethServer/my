<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  postChangeInfo,
  postVerifyEmailChange,
  VerifyEmailChangeSchema,
  type VerifyEmailChange,
} from '@/lib/account'
import { getBackendErrorMessage, getValidationIssues } from '@/lib/validation'
import { useNotificationsStore } from '@/stores/notifications'
import { focusElement, NeInlineNotification, NeModal, NeTextInput } from '@nethesis/vue-components'
import { useMutation } from '@pinia/colada'
import type { AxiosError } from 'axios'
import { computed, ref, useTemplateRef } from 'vue'
import { useI18n } from 'vue-i18n'
import * as v from 'valibot'

// The second step of an email change: the code mailed to the new address is
// typed here. The address itself is never sent again, the backend applies the
// one it parked, so the modal only needs to know it for the message.
const { visible = false, pendingEmail = '' } = defineProps<{
  visible: boolean
  pendingEmail: string
}>()

const emit = defineEmits(['close', 'verified'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const code = ref('')
const codeRef = useTemplateRef<HTMLInputElement>('codeRef')
const validationIssues = ref<Record<string, string[]>>({})

const {
  mutate: verifyMutate,
  isLoading: verifyLoading,
  reset: verifyReset,
  error: verifyError,
} = useMutation({
  mutation: (payload: VerifyEmailChange) => {
    return postVerifyEmailChange(payload)
  },
  onSuccess() {
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('account.email_updated'),
        description: t('account.email_updated_description', { email: pendingEmail }),
      })
    }, 500)
    emit('verified')
  },
  onError: (error) => {
    console.error('Error verifying email change:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'account')
    code.value = ''
    focusElement(codeRef)
  },
})

const {
  mutate: resendMutate,
  isLoading: resendLoading,
  reset: resendReset,
  error: resendError,
} = useMutation({
  mutation: () => {
    return postChangeInfo({ email: pendingEmail })
  },
  onSuccess() {
    notificationsStore.createNotification({
      kind: 'success',
      title: t('account.verification_code_sent'),
      description: t('account.verification_code_sent_description', { email: pendingEmail }),
    })
  },
  onError: (error) => {
    console.error('Error resending verification code:', error)
  },
})

// A wrong code is reported under the field; anything else (no pending change
// left, address taken by another account, backend down) as a notification.
const generalError = computed(() => {
  if (verifyError.value && !validationIssues.value.code) {
    return getBackendErrorMessage(verifyError.value)
  }
  if (resendError.value) {
    return getBackendErrorMessage(resendError.value)
  }
  return ''
})

function onShow() {
  clearErrors()
  code.value = ''
  focusElement(codeRef)
}

function clearErrors() {
  validationIssues.value = {}
  verifyReset()
  resendReset()
}

function validate(payload: VerifyEmailChange): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(VerifyEmailChangeSchema, payload)

  if (validation.success) {
    return true
  }
  const issues = v.flatten(validation.issues)
  if (issues.nested) {
    validationIssues.value = issues.nested as Record<string, string[]>
    focusElement(codeRef)
  }
  return false
}

function verify() {
  clearErrors()

  const payload = { code: code.value.trim() }
  if (!validate(payload)) {
    return
  }
  verifyMutate(payload)
}

function resend() {
  clearErrors()
  resendMutate()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('account.verify_email')"
    kind="info"
    :primary-label="$t('account.verify')"
    :secondary-label="$t('account.resend_code')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="verifyLoading || resendLoading"
    :primary-button-loading="verifyLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="verify"
    @secondary-click="resend"
    @show="onShow"
  >
    <form class="flex flex-col gap-6" @submit.prevent="verify">
      <p>{{ t('account.verify_email_description', { email: pendingEmail }) }}</p>
      <NeTextInput
        ref="codeRef"
        v-model="code"
        :label="$t('account.verification_code')"
        :invalid-message="validationIssues.code?.[0] ? $t(validationIssues.code[0]) : ''"
        :disabled="verifyLoading || resendLoading"
        autocomplete="one-time-code"
        inputmode="numeric"
      />
      <NeInlineNotification
        v-if="generalError"
        kind="error"
        :title="t('account.cannot_verify_email')"
        :description="generalError"
      />
    </form>
  </NeModal>
</template>
