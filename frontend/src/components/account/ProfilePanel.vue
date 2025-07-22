<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang="ts" setup>
import { getMe } from '@/lib/me'
import { putUser, type EditUser } from '@/lib/users'
import { getValidationIssues } from '@/lib/validation'
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import { NeButton, NeInlineNotification, NeSkeleton, NeTextInput } from '@nethesis/vue-components'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import type { AxiosError } from 'axios'
import { ref, useTemplateRef, watch, type ShallowRef } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()
const { state: me, asyncStatus: meAsyncStatus } = useQuery({
  key: ['me'], //// use key factory?
  enabled: () => !!loginStore.jwtToken,
  query: getMe,
})

const {
  mutate: editUserMutate,
  isLoading: editUserLoading,
  reset: editUserReset,
  error: editUserError,
} = useMutation({
  mutation: (user: EditUser) => {
    return putUser(user)
  },
  onSuccess(data, vars, context) {
    notificationsStore.createNotification({
      kind: 'success',
      title: t('account.profile_saved'),
    })
  },
  onError: (error, variables) => {
    console.error('Error editing user:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'users')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: ['me'] })
    queryCache.invalidateQueries({ key: ['users'] })
  },
})

//// loading indicator?

const name = ref('')
const nameRef = useTemplateRef<HTMLInputElement>('nameRef')
const email = ref('')
const phone = ref('')
const phoneRef = useTemplateRef<HTMLInputElement>('phoneRef')
const validationIssues = ref<Record<string, string[]>>({})
const queryCache = useQueryCache()

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  name: nameRef,
  phone: phoneRef,
}

watch(
  () => me.value.data,
  () => {
    if (me.value.data) {
      name.value = me.value.data.name || ''
      email.value = me.value.data.email || ''
      // phone.value = me.value.data.phone || '' //// uncomment
    }
  },
)

function clearErrors() {
  editUserReset()
  validationIssues.value = {}
}

async function saveProfile() {
  clearErrors()

  if (loginStore.userInfo?.id) {
    const user = {
      id: loginStore.userInfo?.id,
      name: name.value,
    }

    //// TODO uncomment!
    // const isValidationOk = validateEdit(userToEdit)
    // if (!isValidationOk) {
    //   return
    // }
    editUserMutate(user as EditUser) //// remove casting
  }
}
</script>

<template>
  <div>
    <!-- get me error notification -->
    <NeInlineNotification
      v-if="me.status === 'error'"
      kind="error"
      :title="$t('account.cannot_retrieve_profile_data')"
      :description="me.error.message"
      class="mb-6"
    />
    <NeSkeleton v-if="me.status === 'pending'" :lines="7" class="w-full" />
    <template v-else>
      <form @submit.prevent class="space-y-6">
        <!-- name -->
        <NeTextInput
          ref="nameRef"
          v-model.trim="name"
          :label="$t('users.name')"
          :invalid-message="validationIssues.name?.[0] ? $t(validationIssues.name[0]) : ''"
          :disabled="editUserLoading"
        />
        <!-- email -->
        <NeTextInput
          ref="emailRef"
          v-model.trim="email"
          :label="$t('users.email')"
          :invalid-message="validationIssues.email?.[0] ? $t(validationIssues.email[0]) : ''"
          disabled
        />
        <!-- save button -->
        <NeButton
          type="submit"
          kind="primary"
          size="lg"
          :disabled="editUserLoading"
          :loading="editUserLoading"
          @click.prevent="saveProfile"
        >
          {{ $t('account.save_profile') }}
        </NeButton>
      </form>
    </template>
  </div>
</template>
