<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { deleteCustomer, type Customer } from '@/lib/customers'
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
  onError: (error, variables) => {
    console.error('Error deleting customer:', error)
  },

  onSettled: () => queryCache.invalidateQueries({ key: ['customers'] }),
})

function onShow() {
  // clear error
  deleteCustomerReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('customers.delete_customer')"
    kind="warning"
    :primary-label="$t('common.delete')"
    :cancel-label="$t('common.cancel')"
    primary-button-kind="danger"
    :primary-button-disabled="deleteCustomerLoading"
    :primary-button-loading="deleteCustomerLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="deleteCustomerMutate(customer!)"
    @show="onShow"
  >
    <p>
      {{ t('customers.delete_customer_confirmation', { name: customer?.name }) }}
    </p>
    <NeInlineNotification
      v-if="deleteCustomerError?.message"
      kind="error"
      :title="t('customers.cannot_delete_customer')"
      :description="deleteCustomerError.message"
      class="mt-4"
    />
  </NeModal>
</template>
