<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import {
  deleteDistributor,
  DISTRIBUTORS_KEY,
  type Distributor,
} from '@/lib/organizations/distributors'
import { useNotificationsStore } from '@/stores/notifications'
import DeleteObjectModal from '../DeleteObjectModal.vue'

const { visible = false, distributor = undefined } = defineProps<{
  visible: boolean
  distributor: Distributor | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: deleteDistributorMutate,
  isLoading: deleteDistributorLoading,
  reset: deleteDistributorReset,
  error: deleteDistributorError,
} = useMutation({
  mutation: (distributor: Distributor) => {
    return deleteDistributor(distributor)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('distributors.distributor_archived'),
        description: t('common.object_archived_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error deleting distributor:', error)
  },
  onSettled: () => queryCache.invalidateQueries({ key: [DISTRIBUTORS_KEY] }),
})

function onShow() {
  // clear error
  deleteDistributorReset()
}
</script>

<template>
  <DeleteObjectModal
    :visible="visible"
    :title="$t('distributors.archive_distributor')"
    :primary-label="$t('common.archive')"
    :deleting="deleteDistributorLoading"
    :confirmation-message="
      t('distributors.archive_distributor_confirmation', { name: distributor?.name })
    "
    :confirmation-input="distributor?.name"
    :error-title="t('organizations.cannot_archive_organization')"
    :error-description="deleteDistributorError?.message"
    @show="onShow"
    @close="emit('close')"
    @primary-click="deleteDistributorMutate(distributor!)"
  />
</template>
