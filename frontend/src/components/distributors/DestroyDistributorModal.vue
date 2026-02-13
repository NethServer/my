<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import {
  destroyDistributor,
  DISTRIBUTORS_KEY,
  DISTRIBUTORS_TOTAL_KEY,
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
  mutate: destroyDistributorMutate,
  isLoading: destroyDistributorLoading,
  reset: destroyDistributorReset,
  error: destroyDistributorError,
} = useMutation({
  mutation: (distributor: Distributor) => {
    return destroyDistributor(distributor)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('distributors.distributor_destroyed'),
        description: t('common.object_destroyed_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error destroying distributor:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_KEY] })
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  destroyDistributorReset()
}
</script>

<template>
  <DeleteObjectModal
    :visible="visible"
    :title="$t('distributors.destroy_distributor')"
    :primary-label="$t('common.destroy')"
    :deleting="destroyDistributorLoading"
    :confirmation-message="
      t('distributors.destroy_distributor_confirmation', { name: distributor?.name })
    "
    :confirmation-input="distributor?.name"
    :error-title="t('organizations.cannot_destroy_organization')"
    :error-description="destroyDistributorError?.message"
    @show="onShow"
    @close="emit('close')"
    @primary-click="destroyDistributorMutate(distributor!)"
  />
</template>
