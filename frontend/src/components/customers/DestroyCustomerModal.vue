<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import {
  destroyCustomer,
  CUSTOMERS_KEY,
  CUSTOMERS_TOTAL_KEY,
  type Customer,
} from '@/lib/organizations/customers'
import { useNotificationsStore } from '@/stores/notifications'
import DeleteObjectModal from '../DeleteObjectModal.vue'

const { visible = false, customer = undefined } = defineProps<{
  visible: boolean
  customer: Customer | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const notificationsStore = useNotificationsStore()
const queryCache = useQueryCache()

const {
  mutate: destroyCustomerMutate,
  isLoading: destroyCustomerLoading,
  reset: destroyCustomerReset,
  error: destroyCustomerError,
} = useMutation({
  mutation: (customer: Customer) => {
    return destroyCustomer(customer)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('customers.customer_destroyed'),
        description: t('common.object_destroyed_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error destroying customer:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [CUSTOMERS_KEY] })
    queryCache.invalidateQueries({ key: [CUSTOMERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  destroyCustomerReset()
}
</script>

<template>
  <DeleteObjectModal
    :visible="visible"
    :title="$t('customers.destroy_customer')"
    :primary-label="$t('common.destroy')"
    :deleting="destroyCustomerLoading"
    :confirmation-message="t('customers.destroy_customer_confirmation', { name: customer?.name })"
    :confirmation-input="customer?.name"
    :error-title="t('organizations.cannot_destroy_organization')"
    :error-description="destroyCustomerError?.message"
    @show="onShow"
    @close="emit('close')"
    @primary-click="destroyCustomerMutate(customer!)"
  />
</template>
