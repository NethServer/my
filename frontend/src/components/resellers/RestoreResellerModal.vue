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
  restoreReseller,
  RESELLERS_KEY,
  RESELLERS_TOTAL_KEY,
  type Reseller,
} from '@/lib/organizations/resellers'
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
  mutate: restoreResellerMutate,
  isLoading: restoreResellerLoading,
  reset: restoreResellerReset,
  error: restoreResellerError,
} = useMutation({
  mutation: (reseller: Reseller) => {
    return restoreReseller(reseller)
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
    console.error('Error restoring reseller:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [RESELLERS_KEY] })
    queryCache.invalidateQueries({ key: [RESELLERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  restoreResellerReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('organizations.restore_organization')"
    kind="warning"
    :primary-label="$t('common.restore')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="restoreResellerLoading"
    :primary-button-loading="restoreResellerLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="restoreResellerMutate(reseller!)"
    @show="onShow"
  >
    <p>
      {{ t('organizations.restore_organization_confirmation', { name: reseller?.name }) }}
    </p>
    <NeInlineNotification
      v-if="restoreResellerError?.message"
      kind="error"
      :title="t('organizations.cannot_restore_organization')"
      :description="restoreResellerError.message"
      class="mt-4"
    />
  </NeModal>
</template>
