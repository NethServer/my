<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { deleteAvatar } from '@/lib/account'
import { USERS_KEY } from '@/lib/users/users'
import { useNotificationsStore } from '@/stores/notifications'
import { useLoginStore } from '@/stores/login'
import { NeInlineNotification, NeModal } from '@nethesis/vue-components'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useI18n } from 'vue-i18n'

const { visible = false } = defineProps<{
  visible: boolean
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const loginStore = useLoginStore()
const queryCache = useQueryCache()

const {
  mutate: deleteAvatarMutate,
  isLoading: deleteAvatarLoading,
  reset: deleteAvatarReset,
  error: deleteAvatarError,
} = useMutation({
  mutation: () => {
    return deleteAvatar()
  },
  onSuccess() {
    loginStore.refreshAvatar()

    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('account.profile_picture_removed'),
        description: t('account.profile_picture_removed_description'),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error deleting avatar:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [USERS_KEY] })
  },
})

function onShow() {
  deleteAvatarReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('account.remove_picture')"
    kind="warning"
    :primary-label="$t('common.remove')"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="primary"
    :primary-button-disabled="deleteAvatarLoading"
    :primary-button-loading="deleteAvatarLoading"
    :close-aria-label="$t('common.close')"
    @show="onShow"
    @close="emit('close')"
    @primary-click="deleteAvatarMutate()"
  >
    <p>{{ t('account.remove_picture_confirmation') }}</p>
    <NeInlineNotification
      v-if="deleteAvatarError?.message"
      kind="error"
      :title="t('account.cannot_remove_profile_picture')"
      :description="deleteAvatarError.message"
      class="mt-4"
    />
  </NeModal>
</template>
