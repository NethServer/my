<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang="ts" setup>
import { ProfileInfoSchema, postChangeInfo, type ProfileInfo } from '@/lib/account'
import { API_URL } from '@/lib/config'
import { getValidationIssues, isValidationError } from '@/lib/validation'
import { useLoginStore } from '@/stores/login'
import { useNotificationsStore } from '@/stores/notifications'
import { faCircleXmark, faPenToSquare } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  NeButton,
  NeCard,
  NeDropdown,
  NeFormItemLabel,
  NeInlineNotification,
  NeSkeleton,
  NeTextInput,
} from '@nethesis/vue-components'
import { useMutation, useQueryCache } from '@pinia/colada'
import axios, { type AxiosError } from 'axios'
import { ref, useTemplateRef, watch, type ShallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import * as v from 'valibot'
import { USERS_KEY } from '@/lib/users/users'
import UserRoleBadge from '../users/UserRoleBadge.vue'
import UserAvatar from '../users/UserAvatar.vue'
import ChangePictureDrawer from './ChangePictureDrawer.vue'
import RemoveAvatarModal from './RemoveAvatarModal.vue'

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
    queryCache.invalidateQueries({ key: [USERS_KEY] })
  },
})

const isChangePictureDrawerShown = ref(false)
const isRemoveAvatarModalShown = ref(false)
const hasCustomAvatar = ref(false)
const loadingCustomAvatar = ref(false)
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

watch(
  [() => loginStore.userInfo?.logto_id, () => loginStore.avatarVersion],
  async ([logtoId]) => {
    if (!logtoId) {
      hasCustomAvatar.value = false
      return
    }

    loadingCustomAvatar.value = true

    try {
      const avatarResponse = await axios.get(`${API_URL}/public/users/${logtoId}/avatar`, {
        validateStatus: (status) => status === 200 || status === 404,
      })
      hasCustomAvatar.value = avatarResponse.status === 200
    } catch (error) {
      console.error('Error checking avatar:', error)
      hasCustomAvatar.value = false
    } finally {
      loadingCustomAvatar.value = false
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
      fieldRefs[firstErrorFieldName].value?.focus()
    }
    return false
  }
}

function getKebabMenuItems() {
  return [
    {
      id: 'removePicture',
      label: t('account.remove_picture'),
      icon: faCircleXmark,
      action: () => {
        isRemoveAvatarModalShown.value = true
      },
      disabled: loadingCustomAvatar.value || !hasCustomAvatar.value,
    },
  ]
}
</script>

<template>
  <div>
    <NeSkeleton v-if="loginStore.loadingUserInfo || editUserLoading" :lines="12" class="w-full" />
    <form v-else @submit.prevent class="space-y-7">
      <!-- avatar -->
      <NeCard>
        <div class="flex items-center justify-between gap-4">
          <UserAvatar
            size="3xl"
            :name="loginStore.userDisplayName"
            :is-owner="loginStore.isOwner"
            :logto-id="loginStore.userInfo?.logto_id || ''"
            :cache-key="loginStore.avatarVersion"
          />
          <div class="flex shrink-0 items-center gap-2">
            <NeButton
              kind="secondary"
              type="button"
              size="lg"
              :disabled="loginStore.isOwner || loginStore.isImpersonating"
              @click="isChangePictureDrawerShown = true"
            >
              <template #prefix>
                <FontAwesomeIcon :icon="faPenToSquare" class="size-4" aria-hidden="true" />
              </template>
              {{ $t('account.change_picture') }}
            </NeButton>
            <!-- kebab menu -->
            <NeDropdown
              :items="getKebabMenuItems()"
              :align-to-right="true"
              :disabled="
                loginStore.isOwner ||
                loginStore.isImpersonating ||
                loadingCustomAvatar ||
                !hasCustomAvatar
              "
            />
          </div>
        </div>
      </NeCard>

      <!-- name -->
      <NeTextInput
        ref="nameRef"
        v-model.trim="name"
        :label="$t('users.name')"
        :invalid-message="validationIssues.name?.[0] ? $t(validationIssues.name[0]) : ''"
        :disabled="editUserLoading || loginStore.isOwner || loginStore.isImpersonating"
      />
      <!-- email -->
      <NeTextInput
        ref="emailRef"
        v-model.trim="email"
        :label="$t('users.email')"
        :invalid-message="validationIssues.email?.[0] ? $t(validationIssues.email[0]) : ''"
        :disabled="editUserLoading || loginStore.isOwner || loginStore.isImpersonating"
      />
      <!-- phone -->
      <NeTextInput
        ref="phoneRef"
        v-model.trim="phone"
        :label="$t('users.phone_number')"
        :invalid-message="validationIssues.phone?.[0] ? $t(validationIssues.phone[0]) : ''"
        :disabled="editUserLoading || loginStore.isOwner || loginStore.isImpersonating"
        :optional="true"
        :optional-label="t('common.optional')"
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
          <UserRoleBadge
            v-for="role in loginStore.userInfo?.user_roles.sort()"
            :key="role"
            :role="role"
          />
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
        :disabled="editUserLoading || loginStore.isOwner || loginStore.isImpersonating"
        :loading="editUserLoading"
        @click.prevent="saveProfile"
      >
        {{ $t('account.save_profile') }}
      </NeButton>
    </form>
  </div>
  <!-- change picture drawer -->
  <ChangePictureDrawer
    :is-shown="isChangePictureDrawerShown"
    @close="isChangePictureDrawerShown = false"
  />
  <RemoveAvatarModal
    :visible="isRemoveAvatarModalShown"
    @close="isRemoveAvatarModalShown = false"
  />
</template>
