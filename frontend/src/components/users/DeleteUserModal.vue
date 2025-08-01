<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { deleteUser, USERS_KEY, USERS_TOTAL_KEY, type User } from '@/lib/users'
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
  mutate: deleteAccountMutate,
  isLoading: deleteAccountLoading,
  reset: deleteAccountReset,
  error: deleteAccountError,
} = useMutation({
  mutation: (user: User) => {
    return deleteUser(user)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('users.user_deleted'),
        description: t('common.object_deleted_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error deleting user:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [USERS_KEY] })
    queryCache.invalidateQueries({ key: [USERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  deleteAccountReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('users.delete_user')"
    kind="warning"
    :primary-label="$t('common.delete')"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="danger"
    :primary-button-disabled="deleteAccountLoading"
    :primary-button-loading="deleteAccountLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="deleteAccountMutate(user!)"
    @show="onShow"
  >
    <p>
      {{ t('users.delete_user_confirmation', { name: user?.name }) }}
    </p>
    <NeInlineNotification
      v-if="deleteAccountError?.message"
      kind="error"
      :title="t('users.cannot_delete_user')"
      :description="deleteAccountError.message"
      class="mt-4"
    />
  </NeModal>
</template>
