<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang="ts" setup>
import { ProfileInfoSchema, postChangeInfo, type ProfileInfo } from '@/lib/account'
import { getValidationIssues, isValidationError } from '@/lib/validation'
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import {
  NeBadge,
  NeButton,
  NeFormItemLabel,
  NeInlineNotification,
  NeTextInput,
} from '@nethesis/vue-components'
import { useMutation, useQueryCache } from '@pinia/colada'
import type { AxiosError } from 'axios'
import { ref, useTemplateRef, watch, type ShallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import * as v from 'valibot'

const { t } = useI18n()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()

const {
  mutate: editUserMutate,
  isLoading: editUserLoading,
  reset: editUserReset,
  error: editUserError,
} = useMutation({
  mutation: (profile: ProfileInfo) => {
    return postChangeInfo(profile)
  },
  onSuccess() {
    notificationsStore.createNotification({
      kind: 'success',
      title: t('account.profile_saved'),
    })

    loginStore.fetchTokenAndUserInfo()
  },
  onError: (error) => {
    console.error('Error editing user:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'users')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: ['authMe'] })
    queryCache.invalidateQueries({ key: ['users'] })
  },
})

const name = ref('')
const nameRef = useTemplateRef<HTMLInputElement>('nameRef')
const email = ref('')
const emailRef = useTemplateRef<HTMLInputElement>('emailRef')
const phone = ref('')
const phoneRef = useTemplateRef<HTMLInputElement>('phoneRef')
const validationIssues = ref<Record<string, string[]>>({})
const queryCache = useQueryCache()

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  name: nameRef,
  email: emailRef,
  phone: phoneRef,
}

watch(
  () => loginStore.userInfo,
  (userInfo) => {
    if (userInfo) {
      name.value = userInfo.name || ''
      email.value = userInfo.email || ''
      phone.value = userInfo.phone || ''
    }
  },
  { immediate: true },
)

function clearErrors() {
  editUserReset()
  validationIssues.value = {}
}

async function saveProfile() {
  clearErrors()

  if (loginStore.userInfo?.id) {
    const profile = {
      name: name.value,
      email: email.value,
      phone: phone.value,
    }

    const isValidationOk = validate(profile)
    if (!isValidationOk) {
      return
    }
    editUserMutate(profile)
  }
}

function validate(profile: ProfileInfo): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(ProfileInfoSchema, profile)

  if (validation.success) {
    // no validation issues
    return true
  } else {
    const issues = v.flatten(validation.issues)

    if (issues.nested) {
      validationIssues.value = issues.nested as Record<string, string[]>

      // focus the first field with error

      const firstErrorFieldName = Object.keys(validationIssues.value)[0]

      console.log('firstFieldName', firstErrorFieldName) ////

      fieldRefs[firstErrorFieldName].value?.focus()
    }
    return false
  }
}
</script>

<template>
  <div>
    <form @submit.prevent class="space-y-7">
      <!-- name -->
      <NeTextInput
        ref="nameRef"
        v-model.trim="name"
        :label="$t('users.name')"
        :invalid-message="validationIssues.name?.[0] ? $t(validationIssues.name[0]) : ''"
        :disabled="editUserLoading || loginStore.isOwner"
      />
      <!-- email -->
      <NeTextInput
        ref="emailRef"
        v-model.trim="email"
        :label="$t('users.email')"
        :invalid-message="validationIssues.email?.[0] ? $t(validationIssues.email[0]) : ''"
        :disabled="editUserLoading || loginStore.isOwner"
      />
      <!-- phone -->
      <NeTextInput
        ref="phoneRef"
        v-model.trim="phone"
        :label="$t('users.phone_number')"
        :invalid-message="validationIssues.phone?.[0] ? $t(validationIssues.phone[0]) : ''"
        :disabled="editUserLoading || loginStore.isOwner"
      />
      <!-- organization -->
      <div>
        <NeFormItemLabel>
          {{ $t('users.organization') }}
        </NeFormItemLabel>
        <div>
          <span>{{ loginStore.userInfo?.organization_name || '-' }}</span>
          <span v-if="loginStore.userInfo?.org_role"> ({{ loginStore.userInfo?.org_role }})</span>
        </div>
      </div>
      <!-- roles -->
      <div>
        <NeFormItemLabel>
          {{ $t('users.roles') }}
        </NeFormItemLabel>
        <div class="flex flex-wrap gap-1">
          <NeBadge
            v-for="role in loginStore.userInfo?.user_roles.sort()"
            :key="role"
            :text="t(`user_roles.${role}`)"
            kind="custom"
            customColorClasses="bg-indigo-100 text-indigo-800 dark:bg-indigo-700 dark:text-indigo-100"
            class="inline-block"
          ></NeBadge>
        </div>
      </div>
      <!-- edit user error notification -->
      <NeInlineNotification
        v-if="editUserError?.message && !isValidationError(editUserError)"
        kind="error"
        :title="t('account.cannot_save_profile_data')"
        :description="editUserError.message"
      />
      <!-- save button -->
      <NeButton
        type="submit"
        kind="primary"
        size="lg"
        :disabled="editUserLoading || loginStore.isOwner"
        :loading="editUserLoading"
        @click.prevent="saveProfile"
      >
        {{ $t('account.save_profile') }}
      </NeButton>
    </form>
  </div>
</template>
