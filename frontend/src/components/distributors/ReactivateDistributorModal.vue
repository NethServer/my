<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import {
  reactivateDistributor,
  DISTRIBUTORS_KEY,
  DISTRIBUTORS_TOTAL_KEY,
  type Distributor,
} from '@/lib/organizations/distributors'
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
  mutate: reactivateDistributorMutate,
  isLoading: reactivateDistributorLoading,
  reset: reactivateDistributorReset,
  error: reactivateDistributorError,
} = useMutation({
  mutation: (distributor: Distributor) => {
    return reactivateDistributor(distributor)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('organizations.organization_reactivated'),
        description: t('organizations.organization_reactivated_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error reactivating distributor:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_KEY] })
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  reactivateDistributorReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('organizations.reactivate_organization')"
    kind="warning"
    :primary-label="$t('common.reactivate')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="reactivateDistributorLoading"
    :primary-button-loading="reactivateDistributorLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="reactivateDistributorMutate(distributor!)"
    @show="onShow"
  >
    <p>
      {{ t('organizations.reactivate_organization_confirmation', { name: distributor?.name }) }}
    </p>
    <NeInlineNotification
      v-if="reactivateDistributorError?.message"
      kind="error"
      :title="t('organizations.cannot_reactivate_organization')"
      :description="reactivateDistributorError.message"
      class="mt-4"
    />
  </NeModal>
</template>
