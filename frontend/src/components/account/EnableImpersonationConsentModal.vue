<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { focusElement, NeInlineNotification, NeTextInput } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import {
  IMPERSONATION_CONSENT_KEY,
  postConsent,
  PostConsentSchema,
  type PostConsent,
} from '@/lib/impersonation'
import { ref, useTemplateRef, type ShallowRef } from 'vue'
import { getValidationIssues } from '@/lib/validation'
import type { AxiosError } from 'axios'
import * as v from 'valibot'

const { visible = false } = defineProps<{
  visible: boolean
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()
const duration = ref<string>('1')
const durationRef = useTemplateRef<HTMLInputElement>('durationRef')
const validationIssues = ref<Record<string, string[]>>({})

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  duration_hours: durationRef,
}
const {
  mutate: postConsentMutate,
  isLoading: postConsentLoading,
  reset: postConsentReset,
  error: postConsentError,
} = useMutation({
  mutation: (consent: PostConsent) => {
    return postConsent(consent)
  },
  onSuccess(data, vars) {
    // show success notification after the drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('account.impersonation_consent_enabled'),
        description: t(
          'account.impersonation_consent_enabled_description',
          {
            hours: vars.duration_hours,
          },
          vars.duration_hours,
        ),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error enabling impersonation consent:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'account')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [IMPERSONATION_CONSENT_KEY] })
  },
})

function onShow() {
  clearErrors()
  duration.value = '1'
  focusElement(durationRef)
}

function clearErrors() {
  validationIssues.value = {}
  postConsentReset()
}

function validate(consent: PostConsent): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(PostConsentSchema, consent)

  if (validation.success) {
    // no validation issues
    return true
  } else {
    const issues = v.flatten(validation.issues)

    if (issues.nested) {
      validationIssues.value = issues.nested as Record<string, string[]>

      // focus the first field with error

      const firstErrorFieldName = Object.keys(validationIssues.value)[0]
      fieldRefs[firstErrorFieldName].value?.focus()
    }
    return false
  }
}

async function enableConsent() {
  clearErrors()

  const durationHours = Number(duration.value)
  const consent = {
    duration_hours: durationHours,
  }

  const isValidationOk = validate(consent)
  if (!isValidationOk) {
    return
  }
  postConsentMutate(consent)
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('account.consent_to_impersonation')"
    kind="warning"
    :primary-label="$t('account.consent')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="postConsentLoading"
    :primary-button-loading="postConsentLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="enableConsent"
    @show="onShow"
  >
    <div class="flex flex-col gap-6">
      <p>
        {{
          t('account.enable_impersonation_modal_message', {
            superAdmin: $t('user_roles.Super Admin'),
          })
        }}
      </p>
      <NeTextInput
        ref="durationRef"
        v-model.trim="duration"
        :label="$t('account.impersonation_max_duration')"
        :helper-text="$t('account.impersonation_max_duration_helper')"
        :invalid-message="
          validationIssues.duration_hours?.[0] ? $t(validationIssues.duration_hours[0]) : ''
        "
        :disabled="postConsentLoading"
        type="number"
        min="1"
        max="168"
      />
      <NeInlineNotification
        v-if="postConsentError?.message"
        kind="error"
        :title="t('account.cannot_enable_impersonation_consent')"
        :description="postConsentError.message"
        class="mt-4"
      />
    </div>
  </NeModal>
</template>
