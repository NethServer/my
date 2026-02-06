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
  suspendDistributor,
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
  mutate: suspendDistributorMutate,
  isLoading: suspendDistributorLoading,
  reset: suspendDistributorReset,
  error: suspendDistributorError,
} = useMutation({
  mutation: (distributor: Distributor) => {
    return suspendDistributor(distributor)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('organizations.organization_suspended'),
        description: t('organizations.organization_suspended_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error suspending distributor:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_KEY] })
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  suspendDistributorReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('organizations.suspend_organization')"
    kind="warning"
    :primary-label="$t('common.suspend')"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="danger"
    :primary-button-disabled="suspendDistributorLoading"
    :primary-button-loading="suspendDistributorLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="suspendDistributorMutate(distributor!)"
    @show="onShow"
  >
    <p>
      {{ t('organizations.suspend_organization_confirmation', { name: distributor?.name }) }}
    </p>
    <NeInlineNotification
      v-if="suspendDistributorError?.message"
      kind="error"
      :title="t('organizations.cannot_suspend_organization')"
      :description="suspendDistributorError.message"
      class="mt-4"
    />
  </NeModal>
</template>
