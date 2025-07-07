<!--
  Copyright (C) 2024 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeSideDrawer,
  NeTextInput,
  focusElement,
  NeInlineNotification,
} from '@nethesis/vue-components'
import { computed, ref, watch } from 'vue'
import { CustomerSchema, postCustomer, putCustomer, type Customer } from '@/lib/customers'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'

//// review

//// search "host" occurrences

const { isShown = false, currentCustomer = undefined } = defineProps<{
  isShown: boolean
  currentCustomer: Customer | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const {
  mutate: createCustomerMutate,
  isLoading: createCustomerLoading,
  reset: createCustomerReset,
  error: createCustomerError,
} = useMutation({
  mutation: (newCustomer: Customer) => {
    return postCustomer(newCustomer)
  },
  onSuccess(data, vars, context) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('customers.customer_created'),
        description: t('common.object_created_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()

    console.log('customer created', data) ////
    console.log('   vars', vars) ////
    console.log('   context', context) ////
  },
  onError: (error, variables) => {
    ////
    console.error('Error creating customer:', error)
    console.error('   variables:', variables)
  },
  //// use key factory?
  onSettled: () => queryCache.invalidateQueries({ key: ['customers'] }),
})

const {
  mutate: editCustomerMutate,
  // status: createCustomerStatus, ////
  // asyncStatus: editCustomerAsyncStatus, //// dont' use?
  isLoading: editCustomerLoading,
  reset: editCustomerReset,
  error: editCustomerError,
} = useMutation({
  mutation: (customer: Customer) => {
    return putCustomer(customer)
  },
  onSuccess(data, vars, context) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('customers.customer_saved'),
        description: t('common.object_saved_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()

    console.log('customer edited', data) ////
    console.log('   vars', vars) ////
    console.log('   context', context) ////
  },
  onError: (error, variables) => {
    ////
    console.error('Error editing customer:', error)
    console.error('   variables:', variables)
  },
  //// use key factory?
  onSettled: () => queryCache.invalidateQueries({ key: ['customers'] }),
})

const name = ref('')
const nameRef = ref()
const description = ref('')
const descriptionRef = ref()
const validationIssues = ref<Record<string, string[]>>({})
// first invalid field ref
const firstErrorRef = ref()

const saving = computed(() => {
  return createCustomerLoading.value || editCustomerLoading.value
})

watch(
  () => isShown,
  () => {
    if (isShown) {
      clearErrors()
      focusElement(nameRef)

      if (currentCustomer) {
        // editing customer
        name.value = currentCustomer.name
        description.value = currentCustomer.description || ''
        ////
      } else {
        // creating customer, reset form to defaults
        name.value = ''
        description.value = ''
        ////
      }
    }
  },
)

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  createCustomerReset()
  editCustomerReset()
  validationIssues.value = {}
}

function validate(customer: Customer): boolean {
  validationIssues.value = {}
  firstErrorRef.value = null

  const validation = v.safeParse(CustomerSchema, customer)

  if (validation.success) {
    // no validation issues
    return true
  } else {
    const issues = v.flatten(validation.issues)

    if (issues.nested) {
      validationIssues.value = issues.nested as Record<string, string[]>

      // focus the first field with error

      const firstErrorFieldName = Object.keys(validationIssues.value)[0]

      console.log('firstFieldName', firstErrorFieldName) ////

      switch (firstErrorFieldName) {
        case 'name':
          firstErrorRef.value = nameRef
          break
        case 'description':
          firstErrorRef.value = descriptionRef
          break
        //// other fields
      }

      if (firstErrorRef.value) {
        focusElement(firstErrorRef.value)
      }
    }
    return false
  }
}

async function saveCustomer() {
  clearErrors()

  const customer: Customer = {
    name: name.value,
    description: description.value,
  }

  const isValidationOk = validate(customer)
  if (!isValidationOk) {
    return
  }

  // loading.value.saveCustomer = true ////

  ////
  // const payload: any = {
  //   name: name.value,
  //   family: ipVersion.value,
  //   ipaddr: records.value,
  // }

  if (currentCustomer?.id) {
    // editing customer
    customer.id = currentCustomer.id
    editCustomerMutate(customer)
  } else {
    createCustomerMutate(customer)
  }

  // try { ////
  //   await ubusCall('ns.objects', apiMethod, payload)
  //   emit('reloadData')
  //   closeDrawer()
  // } catch (err: any) {
  //   console.error(err)

  //   if (err instanceof ValidationError) {
  //     errorBag.value = err.errorBag
  //   } else {
  //     error.value.saveHostSet = $t(getAxiosErrorMessage(err))
  //     error.value.saveHostSetDetails = err.toString()
  //   }
  // } finally {
  //   loading.value.saveCustomer = false
  // }
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="currentCustomer ? $t('customers.edit_customer') : $t('customers.create_customer')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <!-- name -->
        <NeTextInput
          ref="nameRef"
          v-model.trim="name"
          :label="$t('customers.name')"
          :invalid-message="validationIssues.name?.[0] ? $t(validationIssues.name[0]) : ''"
          :disabled="saving"
        />
        <!-- description -->
        <NeTextInput
          ref="descriptionRef"
          v-model.trim="description"
          :label="$t('customers.description')"
          :invalid-message="
            validationIssues.description?.[0] ? $t(validationIssues.description[0]) : ''
          "
          :disabled="saving"
        />
        <!-- create customer error notification -->
        <NeInlineNotification
          v-if="createCustomerError?.message"
          kind="error"
          :title="t('customers.cannot_create_customer')"
          :description="createCustomerError.message"
        />
        <!-- edit customer error notification -->
        <NeInlineNotification
          v-if="editCustomerError?.message"
          kind="error"
          :title="t('customers.cannot_save_customer')"
          :description="editCustomerError.message"
        />
      </div>
      <!-- footer -->
      <hr class="my-8 border-gray-200 dark:border-gray-700" />
      <div class="flex justify-end">
        <NeButton
          kind="tertiary"
          size="lg"
          :disabled="saving"
          class="mr-3"
          @click.prevent="closeDrawer"
        >
          {{ $t('common.cancel') }}
        </NeButton>
        <NeButton
          type="submit"
          kind="primary"
          size="lg"
          :disabled="saving"
          :loading="saving"
          @click.prevent="saveCustomer"
        >
          {{ currentCustomer ? $t('customers.save_customer') : $t('customers.create_customer') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
