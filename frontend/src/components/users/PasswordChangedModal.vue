<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeFormItemLabel, NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { type User } from '@/lib/users/users'
import { ref } from 'vue'
import { useNotificationsStore } from '@/stores/notifications'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faEye, faEyeSlash } from '@fortawesome/free-solid-svg-icons'

const {
  visible = false,
  user = undefined,
  newPassword = '',
} = defineProps<{
  visible: boolean
  user: User | undefined
  newPassword: string
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const isPasswordShown = ref(false)

const copyCredentials = () => {
  if (user && newPassword) {
    const credentials = `${t('users.email')}: ${user.email}\n${t('users.password')}: ${newPassword}`

    navigator.clipboard
      .writeText(credentials)
      .then(() => {
        notificationsStore.createNotification({
          kind: 'success',
          title: t('users.credentials_copied'),
        })
      })
      .catch((err) => {
        console.error('Failed to copy credentials:', err)
      })
  }
}

function onShow() {
  isPasswordShown.value = false
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('users.the_password_has_been_reset')"
    kind="success"
    :primary-label="$t('users.copy_credentials')"
    :cancel-label="$t('common.close')"
    primary-button-kind="primary"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="copyCredentials"
    @show="onShow"
  >
    <p class="mb-6">{{ t('users.credentials_updated_description', { name: user?.name }) }}:</p>
    <div class="mb-4">
      <NeFormItemLabel class="!mb-1">
        {{ t('users.email') }}
      </NeFormItemLabel>
      <span>{{ user?.email }}</span>
    </div>
    <div>
      <NeFormItemLabel class="!mb-1">
        {{ t('users.password') }}
      </NeFormItemLabel>
      <span v-if="isPasswordShown">
        {{ newPassword }}
      </span>
      <span v-else> ********** </span>
      <NeButton
        kind="tertiary"
        size="sm"
        @click="isPasswordShown = !isPasswordShown"
        :aria-label="isPasswordShown ? t('common.hide') : t('common.show')"
        class="ml-3"
      >
        <template #prefix>
          <FontAwesomeIcon :icon="isPasswordShown ? faEyeSlash : faEye" aria-hidden="true" />
        </template>
        {{ isPasswordShown ? t('common.hide') : t('common.show') }}
      </NeButton>
    </div>
  </NeModal>
</template>
