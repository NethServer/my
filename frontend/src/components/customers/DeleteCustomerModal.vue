<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import {
  CUSTOMERS_KEY,
  CUSTOMERS_TOTAL_KEY,
  deleteCustomer,
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
  mutate: deleteCustomerMutate,
  isLoading: deleteCustomerLoading,
  reset: deleteCustomerReset,
  error: deleteCustomerError,
} = useMutation({
  mutation: (customer: Customer) => {
    return deleteCustomer(customer)
  },
  onSuccess(data, vars) {
    // show success notification after modal closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('customers.customer_deleted'),
        description: t('common.object_deleted_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    emit('close')
  },
  onError: (error) => {
    console.error('Error deleting customer:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [CUSTOMERS_KEY] })
    queryCache.invalidateQueries({ key: [CUSTOMERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  deleteCustomerReset()
}
</script>

<template>
  <DeleteObjectModal
    :visible="visible"
    :title="$t('customers.delete_customer')"
    :deleting="deleteCustomerLoading"
    :confirmation-message="t('customers.delete_customer_confirmation', { name: customer?.name })"
    :confirmation-input="customer?.name"
    :error-title="t('customers.cannot_delete_customer')"
    :error-description="deleteCustomerError?.message"
    @show="onShow"
    @close="emit('close')"
    @primary-click="deleteCustomerMutate(customer!)"
  />
</template>
