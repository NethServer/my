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
  focusElement,
  NeCombobox,
  type NeComboboxOption,
} from '@nethesis/vue-components'
import { computed, ref, useTemplateRef, watch, type Ref, type ShallowRef } from 'vue'
import {
  resetPassword,
  CreateUserSchema,
  EditUserSchema,
  postUser,
  putUser,
  type CreateUser,
  type EditUser,
  type User,
} from '@/lib/users'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { fakerIT as faker } from '@faker-js/faker' ////
import { getValidationIssues, isValidationError } from '../../lib/validation'
import type { AxiosError } from 'axios'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import { getOrganizations } from '@/lib/organizations'
import { getUserRoles } from '@/lib/userRoles'
import { generateRandomPassword } from '@/lib/password'

//// review

const { isShown = false } = defineProps<{
  isShown: boolean
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const loginStore = useLoginStore()

const {
  mutate: changePasswordMutate,
  isLoading: changePasswordLoading,
  reset: changePasswordReset,
  error: changePasswordError,
} = useMutation({
  mutation: (user: User) => {
    return resetPassword(user, newPassword.value)
  },
  onSuccess(data, vars) {
    // show success notification after the drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('users.password_reset'),
        description: t('users.password_reset_description'),
      })
    }, 500)
    emit('close')
  },
  onError: (error) => {
    console.error('Error changing password:', error)
  },
})

const oldPassword = ref('')
const oldPasswordRef = useTemplateRef<HTMLInputElement>('oldPasswordRef')
const newPassword = ref('')
const newPasswordRef = useTemplateRef<HTMLInputElement>('newPasswordRef')
const confirmPassword = ref('')
const confirmPasswordRef = useTemplateRef<HTMLInputElement>('confirmPasswordRef')
const validationIssues = ref<Record<string, string[]>>({})

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  oldPassword: oldPasswordRef,
  newPassword: newPasswordRef,
  confirmPassword: confirmPasswordRef,
}

watch(
  () => isShown,
  () => {
    if (isShown) {
      clearErrors()
      focusElement(oldPasswordRef)
      oldPassword.value = ''
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
}

function validateCreate(user: CreateUser): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(CreateUserSchema, user) ////
  // const validation = { success: true } ////

  if (validation.success) {
    // no validation issues
    return true
  } else {
    const issues = v.flatten(validation.issues)

    if (issues.nested) {
      validationIssues.value = issues.nested as Record<string, string[]>

      console.log('validationIssues', validationIssues.value) ////

      // focus the first field with error

      const firstErrorFieldName = Object.keys(validationIssues.value)[0]

      console.log('firstFieldName', firstErrorFieldName) ////

      fieldRefs[firstErrorFieldName]?.value?.focus()
    }
    return false
  }
}

async function validateAndChangePassword() {
  clearErrors()

  //// TODO
  // const user = {
  //   email: oldPassword.value,
  //   name: name.value,
  //   userRoleIds: userRoleIds.value.map((role) => role.id),
  //   organizationId: newPassword.value,
  //   phone: phone.value.replace(/[\+\s\.\-]/g, ''), // remove formatting characters from phone number
  //   customData: {}, //// TODO
  // }

  // if (currentUser?.id) {
  //   // editing user

  //   const userToEdit: EditUser = {
  //     ...user,
  //     id: currentUser.id,
  //   }

  //   const isValidationOk = validateEdit(userToEdit)
  //   if (!isValidationOk) {
  //     return
  //   }
  //   editUserMutate(userToEdit)
  // } else {
  //   // creating user

  //   const userToCreate: CreateUser = {
  //     ...user,
  //     email: oldPassword.value,
  //     password: password.value,
  //   }

  //   const isValidationOk = validateCreate(userToCreate)
  //   if (!isValidationOk) {
  //     return
  //   }
  //   changePasswordMutate(userToCreate)
  // }
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="$t('account.change_password')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <!-- old password -->
        <NeTextInput
          ref="oldPasswordRef"
          v-model="oldPassword"
          is-password
          auto-complete="old-password"
          :label="$t('account.old_password')"
          :invalid-message="
            validationIssues.oldPassword?.[0] ? $t(validationIssues.oldPassword[0]) : ''
          "
          :disabled="changePasswordLoading"
        />
        <!-- new password -->
        <NeTextInput
          ref="newPasswordRef"
          v-model="newPassword"
          is-password
          auto-complete="new-password"
          :label="$t('account.new_password')"
          :invalid-message="
            validationIssues.newPassword?.[0] ? $t(validationIssues.newPassword[0]) : ''
          "
          :disabled="changePasswordLoading"
        />
        <!-- confirm password -->
        <NeTextInput
          ref="confirmPasswordRef"
          v-model="confirmPassword"
          is-password
          auto-complete="confirm-password"
          :label="$t('account.confirm_password')"
          :invalid-message="validationIssues.confirm?.[0] ? $t(validationIssues.confirm[0]) : ''"
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
          :disabled="changePasswordLoading"
          :loading="changePasswordLoading"
          @click.prevent="validateAndChangePassword"
        >
          {{ $t('account.change_password') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
