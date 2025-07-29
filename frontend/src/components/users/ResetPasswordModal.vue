<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation } from '@pinia/colada'
import { resetPassword, type User } from '@/lib/users'
import { useNotificationsStore } from '@/stores/notifications'
import { generateRandomPassword } from '@/lib/password'
import { ref } from 'vue'

const { visible = false, user = undefined } = defineProps<{
  visible: boolean
  user: User | undefined
}>()

const emit = defineEmits<{
  close: []
  'password-changed': [newPassword: string]
}>()

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const newPassword = ref<string>('')

const {
  mutate: resetPasswordMutate,
  isLoading: resetPasswordLoading,
  reset: resetPasswordReset,
  error: resetPasswordError,
} = useMutation({
  mutation: (user: User) => {
    newPassword.value = generateRandomPassword(12)
    return resetPassword(user, newPassword.value)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('users.password_reset'),
        description: t('users.password_reset_description', {
          name: vars.name,
        }),
      })
    }, 500)
    emit('password-changed', newPassword.value)
    emit('close')
  },
  onError: (error) => {
    console.error('Error resetting password:', error)
  },
})

function onShow() {
  // clear error
  resetPasswordReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('users.reset_password')"
    kind="warning"
    :primary-label="$t('users.reset_password')"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="danger"
    :primary-button-disabled="resetPasswordLoading"
    :primary-button-loading="resetPasswordLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="resetPasswordMutate(user!)"
    @show="onShow"
  >
    <p>
      {{ t('users.reset_password_confirmation', { name: user?.name }) }}
    </p>
    <NeInlineNotification
      v-if="resetPasswordError?.message"
      kind="error"
      :title="t('users.cannot_reset_password')"
      :description="resetPasswordError.message"
      class="mt-4"
    />
  </NeModal>
</template>
