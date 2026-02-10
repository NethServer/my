<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { NeModal } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import { restoreCustomer, CUSTOMERS_KEY, CUSTOMERS_TOTAL_KEY, type Customer } from '@/lib/customers'
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
  mutate: restoreCustomerMutate,
  isLoading: restoreCustomerLoading,
  reset: restoreCustomerReset,
  error: restoreCustomerError,
} = useMutation({
  mutation: (customer: Customer) => {
    return restoreCustomer(customer)
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
    console.error('Error restoring customer:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [CUSTOMERS_KEY] })
    queryCache.invalidateQueries({ key: [CUSTOMERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  restoreCustomerReset()
}
</script>

<template>
  <NeModal
    :visible="visible"
    :title="$t('organizations.restore_organization')"
    kind="warning"
    :primary-label="$t('common.restore')"
    :cancel-label="$t('common.cancel')"
    :primary-button-disabled="restoreCustomerLoading"
    :primary-button-loading="restoreCustomerLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="restoreCustomerMutate(customer!)"
    @show="onShow"
  >
    <p>
      {{ t('organizations.restore_organization_confirmation', { name: customer?.name }) }}
    </p>
    <NeInlineNotification
      v-if="restoreCustomerError?.message"
      kind="error"
      :title="t('organizations.cannot_restore_organization')"
      :description="restoreCustomerError.message"
      class="mt-4"
    />
  </NeModal>
</template>
