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
  suspendCustomer,
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
  mutate: suspendCustomerMutate,
  isLoading: suspendCustomerLoading,
  reset: suspendCustomerReset,
  error: suspendCustomerError,
} = useMutation({
  mutation: (customer: Customer) => {
    return suspendCustomer(customer)
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
    console.error('Error suspending customer:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [CUSTOMERS_KEY] })
    queryCache.invalidateQueries({ key: [CUSTOMERS_TOTAL_KEY] })
  },
})

function onShow() {
  // clear error
  suspendCustomerReset()
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
    :primary-button-disabled="suspendCustomerLoading"
    :primary-button-loading="suspendCustomerLoading"
    :close-aria-label="$t('common.close')"
    @close="emit('close')"
    @primary-click="suspendCustomerMutate(customer!)"
    @show="onShow"
  >
    <p>
      {{ t('organizations.suspend_organization_confirmation', { name: customer?.name }) }}
    </p>
    <NeInlineNotification
      v-if="suspendCustomerError?.message"
      kind="error"
      :title="t('organizations.cannot_suspend_organization')"
      :description="suspendCustomerError.message"
      class="mt-4"
    />
  </NeModal>
</template>
