<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useNotificationsStore } from '@/stores/notifications'
import { useLoginStore } from '@/stores/login'
import { ref } from 'vue'
import { isValidationError } from '@/lib/validation'
import type { User } from '@/lib/users'

const { visible = false, user = undefined } = defineProps<{
  visible: boolean
  user: User | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const loginStore = useLoginStore()
const notificationsStore = useNotificationsStore()
const loadingImpersonate = ref(false)
const errorImpersonate = ref<Error | null>(null)

async function impersonateUser(user: User) {
  loadingImpersonate.value = true
  errorImpersonate.value = null

  loginStore
    .impersonateUser(user.logto_id!)
    .then(() => {
      // show success notification after modal closes
      setTimeout(() => {
        notificationsStore.createNotification({
          kind: 'success',
          title: t('users.impersonation_started'),
          description: t('users.you_are_now_impersonating_user', {
            user: user.name,
          }),
        })
      }, 500)

      emit('close')
    })
    .catch((error: Error) => {
      errorImpersonate.value = error
    })
    .finally(() => {
      loadingImpersonate.value = false
    })
}

function onShow() {
  // clear error
  errorImpersonate.value = null
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('users.impersonate_user')"
    kind="warning"
    :primary-label="$t('users.impersonate_user')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="loadingImpersonate"
    :primary-button-loading="loadingImpersonate"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="impersonateUser(user!)"
    @show="onShow"
  >
    <p>
      {{ t('users.impersonate_user_message_1', { user: user?.name }) }}
    </p>
    <p class="mt-2">
      {{ t('users.impersonate_user_message_2') }}
    </p>
    <NeInlineNotification
      v-if="errorImpersonate?.message && !isValidationError(errorImpersonate)"
      kind="error"
      :title="t('users.cannot_impersonate_user')"
      :description="errorImpersonate.message"
      class="mt-4"
    />
  </NeModal>
</template>
