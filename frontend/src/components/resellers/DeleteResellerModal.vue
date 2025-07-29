<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { deleteReseller, type Reseller } from '@/lib/resellers'
import { useNotificationsStore } from '@/stores/notifications'

const { visible = false, reseller = undefined } = defineProps<{
  visible: boolean
  reseller: Reseller | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: deleteResellerMutate,
  isLoading: deleteResellerLoading,
  reset: deleteResellerReset,
  error: deleteResellerError,
} = useMutation({
  mutation: (reseller: Reseller) => {
    return deleteReseller(reseller)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('resellers.reseller_deleted'),
        description: t('common.object_deleted_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error, variables) => {
    console.error('Error deleting reseller:', error)
  },

  onSettled: () => queryCache.invalidateQueries({ key: ['resellers'] }),
})

function onShow() {
  // clear error
  deleteResellerReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('resellers.delete_reseller')"
    kind="warning"
    :primary-label="$t('common.delete')"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="danger"
    :primary-button-disabled="deleteResellerLoading"
    :primary-button-loading="deleteResellerLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="deleteResellerMutate(reseller!)"
    @show="onShow"
  >
    <p>
      {{ t('resellers.delete_reseller_confirmation', { name: reseller?.name }) }}
    </p>
    <NeInlineNotification
      v-if="deleteResellerError?.message"
      kind="error"
      :title="t('resellers.cannot_delete_reseller')"
      :description="deleteResellerError.message"
      class="mt-4"
    />
  </NeModal>
</template>
