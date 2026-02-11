<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { restoreUser, USERS_KEY, USERS_TOTAL_KEY, type User } from '@/lib/users/users'
import { useNotificationsStore } from '@/stores/notifications'

const { visible = false, user = undefined } = defineProps<{
  visible: boolean
  user: User | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: restoreUserMutate,
  isLoading: restoreUserLoading,
  reset: restoreUserReset,
  error: restoreUserError,
} = useMutation({
  mutation: (user: User) => {
    return restoreUser(user)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('users.user_restored'),
        description: t('users.user_restored_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error restoring user:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [USERS_KEY] })
    queryCache.invalidateQueries({ key: [USERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  restoreUserReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('users.restore_user')"
    kind="warning"
    :primary-label="$t('common.restore')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="restoreUserLoading"
    :primary-button-loading="restoreUserLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="restoreUserMutate(user!)"
    @show="onShow"
  >
    <p>
      {{ t('users.restore_user_confirmation', { name: user?.name }) }}
    </p>
    <NeInlineNotification
      v-if="restoreUserError?.message"
      kind="error"
      :title="t('users.cannot_restore_user')"
      :description="restoreUserError.message"
      class="mt-4"
    />
  </NeModal>
</template>
