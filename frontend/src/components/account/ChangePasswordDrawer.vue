<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeSideDrawer,
  NeTextInput,
  NeInlineNotification,
  NePasswordRequirements,
  focusElement,
  usePasswordRequirements,
  type NePasswordRequirement,
} from '@nethesis/vue-components'
import { computed, ref, useTemplateRef, watch, type ShallowRef } from 'vue'
import * as v from 'valibot'
import { useMutation } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { getValidationIssues, isValidationError } from '../../lib/validation'
import { useLoginStore } from '@/stores/login'
import { ChangePasswordSchema, postChangePassword, type ChangePassword } from '@/lib/account'
import {
  PASSWORD_MIN_LENGTH,
  PASSWORD_MAX_LENGTH,
  hasNoWeakPatterns,
  hasSpecialCharacter,
} from '@/lib/password'
import type { AxiosError } from 'axios'
import { useRoute } from 'vue-router'
import router from '@/router'

const { isShown = false } = defineProps<{
  isShown: boolean
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const loginStore = useLoginStore()
const route = useRoute()

const {
  mutate: changePasswordMutate,
  isLoading: changePasswordLoading,
  reset: changePasswordReset,
  error: changePasswordError,
} = useMutation({
  mutation: (changePasswordData: ChangePassword) => {
    return postChangePassword(changePasswordData)
  },
  onSuccess() {
    if (route.query['changePassword'] === 'true') {
      // remove query parameter if it was set to trigger the drawer
      const newQuery = { ...route.query }
      delete newQuery['changePassword']
      router.replace({ query: newQuery })
    }

    // show success notification after the drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('account.password_changed'),
        description: t('account.password_changed_description'),
      })
    }, 500)
    emit('close')
  },
  onError: (error) => {
    console.error('Error changing password:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'account')
  },
})

const currentPassword = ref('')
const currentPasswordRef = useTemplateRef<HTMLInputElement>('currentPasswordRef')
const newPassword = ref('')
const newPasswordRef = useTemplateRef<HTMLInputElement>('newPasswordRef')
const confirmPassword = ref('')
const confirmPasswordRef = useTemplateRef<HTMLInputElement>('confirmPasswordRef')
const validationIssues = ref<Record<string, string[]>>({})
// unmet requirements look pending while the user types, and failed after a submit attempt
const showPasswordRequirementErrors = ref(false)

// the requirements the backend checks, in the order the checklist shows them. The library ships no
// i18n, so every label comes from our translations.
const passwordRequirementDefinitions = computed<NePasswordRequirement[]>(() => [
  {
    key: 'minLength',
    label: t('account.password_requirements.min_length', { minLength: PASSWORD_MIN_LENGTH }),
    validate: (password) => password.length >= PASSWORD_MIN_LENGTH,
  },
  {
    key: 'uppercase',
    label: t('account.password_requirements.uppercase'),
    validate: (password) => /[A-Z]/.test(password),
  },
  {
    key: 'lowercase',
    label: t('account.password_requirements.lowercase'),
    validate: (password) => /[a-z]/.test(password),
  },
  {
    key: 'number',
    label: t('account.password_requirements.number'),
    validate: (password) => /[0-9]/.test(password),
  },
  {
    key: 'specialCharacter',
    label: t('account.password_requirements.special_character'),
    validate: hasSpecialCharacter,
  },
  {
    key: 'weakPatterns',
    label: t('account.password_requirements.no_weak_patterns'),
    validate: hasNoWeakPatterns,
  },
])

const { requirements: passwordRequirements, isValid: isNewPasswordStrongEnough } =
  usePasswordRequirements(newPassword, { requirements: passwordRequirementDefinitions })

const newPasswordInvalidMessage = computed(() => {
  const validationIssue = validationIssues.value.new_password?.[0]

  if (validationIssue) {
    // extra params are ignored by messages that don't use them
    return t(validationIssue, { minLength: PASSWORD_MIN_LENGTH, maxLength: PASSWORD_MAX_LENGTH })
  }

  if (showPasswordRequirementErrors.value && !isNewPasswordStrongEnough.value) {
    return t('account.new_password_requirements_not_met')
  }
  return ''
})

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  currentPassword: currentPasswordRef,
  newPassword: newPasswordRef,
  confirmPassword: confirmPasswordRef,
}

const isFirstAccess = computed(() => {
  return route.query['changePassword'] === 'true'
})

watch(
  () => isShown,
  () => {
    if (isShown) {
      clearErrors()
      focusElement(currentPasswordRef)
      currentPassword.value = ''
      newPassword.value = ''
      confirmPassword.value = ''
    }
  },
)

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  changePasswordReset()
  validationIssues.value = {}
  showPasswordRequirementErrors.value = false
}

function validate(changePasswordData: ChangePassword): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(ChangePasswordSchema, changePasswordData)

  if (validation.success) {
    // no validation issues
    return true
  } else {
    const issues = v.flatten(validation.issues)

    if (issues.nested) {
      validationIssues.value = issues.nested as Record<string, string[]>

      // focus the first field with error

      const firstErrorFieldName = Object.keys(validationIssues.value)[0]
      fieldRefs[firstErrorFieldName]?.value?.focus()
    }
    return false
  }
}

async function validateAndChangePassword() {
  clearErrors()

  const changePasswordData: ChangePassword = {
    current_password: currentPassword.value,
    new_password: newPassword.value,
    confirm_password: confirmPassword.value,
  }

  const isValidationOk = validate(changePasswordData)

  if (!isNewPasswordStrongEnough.value) {
    showPasswordRequirementErrors.value = true

    if (isValidationOk) {
      // no field error stole the focus, point the user at the requirements
      focusElement(newPasswordRef)
    }
    return
  }

  if (!isValidationOk) {
    return
  }
  changePasswordMutate(changePasswordData)
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="$t('account.change_password')"
    :close-aria-label="$t('shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <NeInlineNotification
          v-if="loginStore.isOwner"
          kind="info"
          :title="$t('account.password_change_disabled')"
          :description="$t('account.password_change_disabled_owner_description')"
        />
        <NeInlineNotification
          v-else-if="loginStore.isImpersonating"
          kind="info"
          :title="$t('account.password_change_disabled')"
          :description="$t('account.password_change_disabled_impersonating_description')"
        />
        <NeInlineNotification
          v-else-if="isFirstAccess"
          kind="info"
          :title="$t('account.change_your_temporary_password')"
          :description="$t('account.change_your_temporary_password_description')"
        />
        <!-- current password -->
        <NeTextInput
          ref="currentPasswordRef"
          v-model="currentPassword"
          is-password
          autocomplete="current-password"
          :label="$t('account.current_password')"
          :invalid-message="
            validationIssues.current_password?.[0] ? $t(validationIssues.current_password[0]) : ''
          "
          :disabled="changePasswordLoading"
        />
        <!-- new password -->
        <div class="space-y-3">
          <NeTextInput
            ref="newPasswordRef"
            v-model="newPassword"
            is-password
            autocomplete="new-password"
            :label="$t('account.new_password')"
            :invalid-message="newPasswordInvalidMessage"
            :disabled="changePasswordLoading"
          />
          <NePasswordRequirements
            :requirements="passwordRequirements"
            :show-errors="showPasswordRequirementErrors"
            :met-label="$t('account.password_requirements.requirement_met')"
            :unmet-label="$t('account.password_requirements.requirement_not_met')"
          />
        </div>
        <!-- confirm password -->
        <NeTextInput
          ref="confirmPasswordRef"
          v-model="confirmPassword"
          is-password
          autocomplete="confirm-password"
          :label="$t('account.confirm_password')"
          :invalid-message="
            validationIssues.confirm_password?.[0] ? $t(validationIssues.confirm_password[0]) : ''
          "
          :disabled="changePasswordLoading"
        />
        <!-- change password error notification -->
        <NeInlineNotification
          v-if="changePasswordError?.message && !isValidationError(changePasswordError)"
          kind="error"
          :title="t('account.cannot_change_password')"
          :description="changePasswordError.message"
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
          :disabled="changePasswordLoading || loginStore.isOwner || loginStore.isImpersonating"
          :loading="changePasswordLoading"
          @click.prevent="validateAndChangePassword"
        >
          {{ $t('account.change_password') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
