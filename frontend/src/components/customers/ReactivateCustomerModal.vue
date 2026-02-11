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
  reactivateCustomer,
  CUSTOMERS_KEY,
  CUSTOMERS_TOTAL_KEY,
  type Customer,
} from '@/lib/organizations/customers'
import { useNotificationsStore } from '@/stores/notifications'

const { visible = false, customer = undefined } = defineProps<{
  visible: boolean
  customer: Customer | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: reactivateCustomerMutate,
  isLoading: reactivateCustomerLoading,
  reset: reactivateCustomerReset,
  error: reactivateCustomerError,
} = useMutation({
  mutation: (customer: Customer) => {
    return reactivateCustomer(customer)
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
    console.error('Error reactivating customer:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [CUSTOMERS_KEY] })
    queryCache.invalidateQueries({ key: [CUSTOMERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  reactivateCustomerReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('organizations.reactivate_organization')"
    kind="warning"
    :primary-label="$t('common.reactivate')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="reactivateCustomerLoading"
    :primary-button-loading="reactivateCustomerLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="reactivateCustomerMutate(customer!)"
    @show="onShow"
  >
    <p>
      {{ t('organizations.reactivate_organization_confirmation', { name: customer?.name }) }}
    </p>
    <NeInlineNotification
      v-if="reactivateCustomerError?.message"
      kind="error"
      :title="t('organizations.cannot_reactivate_organization')"
      :description="reactivateCustomerError.message"
      class="mt-4"
    />
  </NeModal>
</template>
