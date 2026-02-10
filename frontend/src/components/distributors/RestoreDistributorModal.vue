<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import {
  restoreDistributor,
  DISTRIBUTORS_KEY,
  DISTRIBUTORS_TOTAL_KEY,
  type Distributor,
} from '@/lib/distributors'
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
  mutate: restoreDistributorMutate,
  isLoading: restoreDistributorLoading,
  reset: restoreDistributorReset,
  error: restoreDistributorError,
} = useMutation({
  mutation: (distributor: Distributor) => {
    return restoreDistributor(distributor)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('organizations.organization_restored'),
        description: t('organizations.organization_restored_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error restoring distributor:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_KEY] })
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  restoreDistributorReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('organizations.restore_organization')"
    kind="warning"
    :primary-label="$t('common.restore')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="restoreDistributorLoading"
    :primary-button-loading="restoreDistributorLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="restoreDistributorMutate(distributor!)"
    @show="onShow"
  >
    <p>
      {{ t('organizations.restore_organization_confirmation', { name: distributor?.name }) }}
    </p>
    <NeInlineNotification
      v-if="restoreDistributorError?.message"
      kind="error"
      :title="t('organizations.cannot_restore_organization')"
      :description="restoreDistributorError.message"
      class="mt-4"
    />
  </NeModal>
</template>
