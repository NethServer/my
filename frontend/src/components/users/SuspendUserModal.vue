<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { suspendUser, USERS_KEY, USERS_TOTAL_KEY, type User } from '@/lib/users'
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
  mutate: suspendUserMutate,
  isLoading: suspendUserLoading,
  reset: suspendUserReset,
  error: suspendUserError,
} = useMutation({
  mutation: (user: User) => {
    return suspendUser(user)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('users.user_suspended'),
        description: t('users.user_suspended_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error suspending user:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [USERS_KEY] })
    queryCache.invalidateQueries({ key: [USERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  suspendUserReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('users.suspend_user')"
    kind="warning"
    :primary-label="$t('common.suspend')"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="danger"
    :primary-button-disabled="suspendUserLoading"
    :primary-button-loading="suspendUserLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="suspendUserMutate(user!)"
    @show="onShow"
  >
    <p>
      {{ t('users.suspend_user_confirmation', { name: user?.name }) }}
    </p>
    <NeInlineNotification
      v-if="suspendUserError?.message"
      kind="error"
      :title="t('users.cannot_suspend_user')"
      :description="suspendUserError.message"
      class="mt-4"
    />
  </NeModal>
</template>
