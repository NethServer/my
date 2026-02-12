<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import {
  deleteReseller,
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
        title: t('resellers.reseller_archived'),
        description: t('common.object_archived_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error deleting reseller:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [RESELLERS_KEY] })
    queryCache.invalidateQueries({ key: [RESELLERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  deleteResellerReset()
}
</script>

<template>
  <DeleteObjectModal
    :visible="visible"
    :title="$t('resellers.archive_reseller')"
    :primary-label="$t('common.archive')"
    :deleting="deleteResellerLoading"
    :confirmation-message="t('resellers.archive_reseller_confirmation', { name: reseller?.name })"
    :confirmation-input="reseller?.name"
    :error-title="t('organizations.cannot_archive_organization')"
    :error-description="deleteResellerError?.message"
    @show="onShow"
    @close="emit('close')"
    @primary-click="deleteResellerMutate(reseller!)"
  />
</template>
