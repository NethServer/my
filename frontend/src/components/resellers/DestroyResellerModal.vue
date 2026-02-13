<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import {
  destroyReseller,
  RESELLERS_KEY,
  RESELLERS_TOTAL_KEY,
  type Reseller,
} from '@/lib/organizations/resellers'
import { useNotificationsStore } from '@/stores/notifications'
import DeleteObjectModal from '../DeleteObjectModal.vue'

const { visible = false, reseller = undefined } = defineProps<{
  visible: boolean
  reseller: Reseller | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: destroyResellerMutate,
  isLoading: destroyResellerLoading,
  reset: destroyResellerReset,
  error: destroyResellerError,
} = useMutation({
  mutation: (reseller: Reseller) => {
    return destroyReseller(reseller)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('resellers.reseller_destroyed'),
        description: t('common.object_destroyed_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error destroying reseller:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [RESELLERS_KEY] })
    queryCache.invalidateQueries({ key: [RESELLERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  destroyResellerReset()
}
</script>

<template>
  <DeleteObjectModal
    :visible="visible"
    :title="$t('resellers.destroy_reseller')"
    :primary-label="$t('common.destroy')"
    :deleting="destroyResellerLoading"
    :confirmation-message="t('resellers.destroy_reseller_confirmation', { name: reseller?.name })"
    :confirmation-input="reseller?.name"
    :error-title="t('organizations.cannot_destroy_organization')"
    :error-description="destroyResellerError?.message"
    @show="onShow"
    @close="emit('close')"
    @primary-click="destroyResellerMutate(reseller!)"
  />
</template>
