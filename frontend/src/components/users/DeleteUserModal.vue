<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { deleteUser, USERS_KEY, USERS_TOTAL_KEY, type User } from '@/lib/users/users'
import { useNotificationsStore } from '@/stores/notifications'
import DeleteObjectModal from '../DeleteObjectModal.vue'

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
        title: t('users.user_archived'),
        description: t('common.object_archived_successfully', {
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
  <DeleteObjectModal
    :visible="visible"
    :title="$t('users.archive_user')"
    :primary-label="$t('common.archive')"
    :deleting="deleteAccountLoading"
    :confirmation-message="t('users.archive_user_confirmation', { name: user?.name })"
    :confirmation-input="user?.name"
    :error-title="t('users.cannot_archive_user')"
    :error-description="deleteAccountError?.message"
    @show="onShow"
    @close="emit('close')"
    @primary-click="deleteAccountMutate(user!)"
  />
</template>
