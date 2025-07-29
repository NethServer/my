<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { deleteDistributor, type Distributor } from '@/lib/distributors'
import { useNotificationsStore } from '@/stores/notifications'

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
        title: t('distributors.distributor_deleted'),
        description: t('common.object_deleted_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error deleting distributor:', error)
  },

  onSettled: () => queryCache.invalidateQueries({ key: ['distributors'] }),
})

function onShow() {
  // clear error
  deleteDistributorReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('distributors.delete_distributor')"
    kind="warning"
    :primary-label="$t('common.delete')"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="danger"
    :primary-button-disabled="deleteDistributorLoading"
    :primary-button-loading="deleteDistributorLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="deleteDistributorMutate(distributor!)"
    @show="onShow"
  >
    <p>
      {{ t('distributors.delete_distributor_confirmation', { name: distributor?.name }) }}
    </p>
    <NeInlineNotification
      v-if="deleteDistributorError?.message"
      kind="error"
      :title="t('distributors.cannot_delete_distributor')"
      :description="deleteDistributorError.message"
      class="mt-4"
    />
  </NeModal>
</template>
