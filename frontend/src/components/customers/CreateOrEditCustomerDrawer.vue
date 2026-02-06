<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeSideDrawer,
  NeTextInput,
  focusElement,
  NeInlineNotification,
  NeTextArea,
} from '@nethesis/vue-components'
import { computed, ref, useTemplateRef, type ShallowRef } from 'vue'
import {
  CreateCustomerSchema,
  CUSTOMERS_KEY,
  CUSTOMERS_TOTAL_KEY,
  CustomerSchema,
  postCustomer,
  putCustomer,
  type CreateCustomer,
  type Customer,
} from '@/lib/customers'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { getValidationIssues, isValidationError } from '@/lib/validation'
import type { AxiosError } from 'axios'

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
  mutation: (newCustomer: CreateCustomer) => {
    return postCustomer(newCustomer)
  },
  onSuccess(data, vars) {
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
  },
  onError: (error) => {
    console.error('Error creating customer:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'organizations')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [CUSTOMERS_KEY] })
    queryCache.invalidateQueries({ key: [CUSTOMERS_TOTAL_KEY] })
  },
})

const {
  mutate: editCustomerMutate,
  isLoading: editCustomerLoading,
  reset: editCustomerReset,
  error: editCustomerError,
} = useMutation({
  mutation: (customer: Customer) => {
    return putCustomer(customer)
  },
  onSuccess(data, vars) {
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
  },
  onError: (error) => {
    console.error('Error editing customer:', error)
  },
  onSettled: () => queryCache.invalidateQueries({ key: [CUSTOMERS_KEY] }),
})

const name = ref('')
const nameRef = useTemplateRef<HTMLInputElement>('nameRef')
const description = ref('')
const descriptionRef = useTemplateRef<HTMLInputElement>('descriptionRef')
const vatNumber = ref('')
const vatNumberRef = useTemplateRef<HTMLInputElement>('vatNumberRef')
const notes = ref('')
const notesRef = useTemplateRef<HTMLInputElement>('notesRef')
const validationIssues = ref<Record<string, string[]>>({})

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  name: nameRef,
  description: descriptionRef,
  custom_data_vat: vatNumberRef,
  custom_data_notes: notesRef,
}

const saving = computed(() => {
  return createCustomerLoading.value || editCustomerLoading.value
})

function onShow() {
  clearErrors()
  focusElement(nameRef)

  if (currentCustomer) {
    // editing customer
    name.value = currentCustomer.name
    description.value = currentCustomer.description || ''
    vatNumber.value = currentCustomer.custom_data?.vat || ''
    notes.value = currentCustomer.custom_data?.notes || ''
  } else {
    // creating customer, reset form to defaults
    name.value = ''
    description.value = ''
    vatNumber.value = ''
    notes.value = ''
  }
}

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  createCustomerReset()
  editCustomerReset()
  validationIssues.value = {}
}

function validateCreate(customer: CreateCustomer): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(CreateCustomerSchema, customer)

  if (validation.success) {
    // no validation issues
    return true
  } else {
    const flattenedIssues = v.flatten(validation.issues)

    if (flattenedIssues.nested) {
      const issues: Record<string, string[]> = {}

      for (const key in flattenedIssues.nested) {
        // replace dots with underscores for i18n key
        const newKey = key.replace(/\./g, '_')
        issues[newKey] = flattenedIssues.nested[key] ?? []
      }
      validationIssues.value = issues

      console.debug('frontend validation issues', validationIssues.value)

      // focus the first field with error

      const firstErrorFieldName = Object.keys(validationIssues.value)[0]
      fieldRefs[firstErrorFieldName]?.value?.focus()
    }
    return false
  }
}

function validateEdit(customer: Customer): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(CustomerSchema, customer)

  if (validation.success) {
    // no validation issues
    return true
  } else {
    const flattenedIssues = v.flatten(validation.issues)

    if (flattenedIssues.nested) {
      const issues: Record<string, string[]> = {}

      for (const key in flattenedIssues.nested) {
        // replace dots with underscores for i18n key
        const newKey = key.replace(/\./g, '_')
        issues[newKey] = flattenedIssues.nested[key] ?? []
      }
      validationIssues.value = issues

      console.debug('frontend validation issues', validationIssues.value)

      // focus the first field with error

      const firstErrorFieldName = Object.keys(validationIssues.value)[0]
      fieldRefs[firstErrorFieldName]?.value?.focus()
    }
    return false
  }
}

async function saveCustomer() {
  clearErrors()

  const customer = {
    name: name.value,
    description: description.value,
    custom_data: {
      vat: vatNumber.value,
      notes: notes.value,
    },
  }

  if (currentCustomer?.logto_id) {
    // editing customer

    const customerToEdit: Customer = {
      ...customer,
      logto_id: currentCustomer.logto_id,
    }

    const isValidationOk = validateEdit(customerToEdit)
    if (!isValidationOk) {
      return
    }
    editCustomerMutate(customerToEdit)
  } else {
    // creating customer

    const customerToCreate: CreateCustomer = customer
    const isValidationOk = validateCreate(customerToCreate)
    if (!isValidationOk) {
      return
    }
    createCustomerMutate(customerToCreate)
  }
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="currentCustomer ? $t('customers.edit_customer') : $t('customers.create_customer')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <!-- name -->
        <NeTextInput
          ref="nameRef"
          v-model="name"
          @blur="name = name.trim()"
          :label="$t('organizations.name')"
          :invalid-message="validationIssues.name?.[0] ? $t(validationIssues.name[0]) : ''"
          :disabled="saving"
        />
        <!-- description -->
        <NeTextInput
          ref="descriptionRef"
          v-model="description"
          @blur="description = description.trim()"
          :label="$t('organizations.description')"
          :invalid-message="
            validationIssues.description?.[0] ? $t(validationIssues.description[0]) : ''
          "
          :disabled="saving"
        />
        <!-- VAT number -->
        <NeTextInput
          ref="vatNumberRef"
          v-model="vatNumber"
          @blur="vatNumber = vatNumber.trim()"
          :label="$t('organizations.vat_number')"
          :invalid-message="
            validationIssues.custom_data_vat?.[0] ? $t(validationIssues.custom_data_vat[0]) : ''
          "
          :disabled="saving"
        />
        <!-- notes -->
        <NeTextArea
          ref="notesRef"
          v-model="notes"
          @blur="notes = notes.trim()"
          :label="$t('organizations.notes')"
          :disabled="saving"
          :invalid-message="validationIssues.notes?.[0] ? $t(validationIssues.notes[0]) : ''"
          :optional="true"
          :optional-label="t('common.optional')"
        />
        <!-- create customer error notification -->
        <NeInlineNotification
          v-if="createCustomerError?.message && !isValidationError(createCustomerError)"
          kind="error"
          :title="t('customers.cannot_create_customer')"
          :description="createCustomerError.message"
        />
        <!-- edit customer error notification -->
        <NeInlineNotification
          v-if="editCustomerError?.message && !isValidationError(editCustomerError)"
          kind="error"
          :title="t('customers.cannot_save_customer')"
          :description="editCustomerError.message"
        />
      </div>
      <!-- footer -->
      <hr class="my-8" />
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
