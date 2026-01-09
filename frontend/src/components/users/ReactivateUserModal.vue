<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { reactivateUser, USERS_KEY, USERS_TOTAL_KEY, type User } from '@/lib/users'
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
  mutate: reactivateUserMutate,
  isLoading: reactivateUserLoading,
  reset: reactivateUserReset,
  error: reactivateUserError,
} = useMutation({
  mutation: (user: User) => {
    return reactivateUser(user)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('users.user_reactivated'),
        description: t('users.user_reactivated_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error reactivating user:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [USERS_KEY] })
    queryCache.invalidateQueries({ key: [USERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  reactivateUserReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('users.reactivate_user')"
    kind="warning"
    :primary-label="$t('common.reactivate')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="reactivateUserLoading"
    :primary-button-loading="reactivateUserLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="reactivateUserMutate(user!)"
    @show="onShow"
  >
    <p>
      {{ t('users.reactivate_user_confirmation', { name: user?.name }) }}
    </p>
    <NeInlineNotification
      v-if="reactivateUserError?.message"
      kind="error"
      :title="t('users.cannot_reactivate_user')"
      :description="reactivateUserError.message"
      class="mt-4"
    />
  </NeModal>
</template>
