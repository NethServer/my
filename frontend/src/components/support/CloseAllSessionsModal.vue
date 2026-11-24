<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification, NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { closeAllSupportSessions, SUPPORT_SESSIONS_KEY } from '@/lib/support/support'
import { useNotificationsStore } from '@/stores/notifications'

const { visible = false } = defineProps<{
  visible: boolean
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: closeAllMutate,
  isLoading: closeAllLoading,
  reset: closeAllReset,
  error: closeAllError,
} = useMutation({
  mutation: () => closeAllSupportSessions(),
  onSuccess(data) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('support.sessions_closed'),
        description: t('support.sessions_closed_count', data.closed),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error closing support sessions:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [SUPPORT_SESSIONS_KEY] })
  },
})

function onShow() {
  // clear error
  closeAllReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('support.close_all_sessions')"
    kind="warning"
    :primary-label="$t('support.close_sessions')"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="danger"
    :primary-button-disabled="closeAllLoading"
    :primary-button-loading="closeAllLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="closeAllMutate()"
    @show="onShow"
  >
    <p>
      {{ $t('support.close_all_sessions_confirmation') }}
    </p>
    <NeInlineNotification
      v-if="closeAllError?.message"
      kind="error"
      :title="$t('support.cannot_close_sessions')"
      :description="closeAllError.message"
      class="mt-4"
    />
  </NeModal>
</template>
