<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { destroyUser, USERS_KEY, USERS_TOTAL_KEY, type User } from '@/lib/users/users'
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
  mutate: destroyUserMutate,
  isLoading: destroyUserLoading,
  reset: destroyUserReset,
  error: destroyUserError,
} = useMutation({
  mutation: (user: User) => {
    return destroyUser(user)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('users.user_destroyed'),
        description: t('common.object_destroyed_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error destroying user:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [USERS_KEY] })
    queryCache.invalidateQueries({ key: [USERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  destroyUserReset()
}
</script>

<template>
  <DeleteObjectModal
    :visible="visible"
    :title="$t('users.destroy_user')"
    :primary-label="$t('common.destroy')"
    :deleting="destroyUserLoading"
    :confirmation-message="t('users.destroy_user_confirmation', { name: user?.name })"
    :confirmation-input="user?.name"
    :error-title="t('users.cannot_destroy_user')"
    :error-description="destroyUserError?.message"
    @show="onShow"
    @close="emit('close')"
    @primary-click="destroyUserMutate(user!)"
  />
</template>
