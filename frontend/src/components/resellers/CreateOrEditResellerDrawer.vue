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
} from '@nethesis/vue-components'
import { computed, ref, useTemplateRef, watch, type ShallowRef } from 'vue'
import {
  CreateResellerSchema,
  RESELLERS_KEY,
  RESELLERS_TOTAL_KEY,
  ResellerSchema,
  postReseller,
  putReseller,
  type CreateReseller,
  type Reseller,
} from '@/lib/resellers'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { getValidationIssues, isValidationError } from '@/lib/validation'
import type { AxiosError } from 'axios'

const { isShown = false, currentReseller = undefined } = defineProps<{
  isShown: boolean
  currentReseller: Reseller | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const {
  mutate: createResellerMutate,
  isLoading: createResellerLoading,
  reset: createResellerReset,
  error: createResellerError,
} = useMutation({
  mutation: (newReseller: CreateReseller) => {
    return postReseller(newReseller)
  },
  onSuccess(data, vars) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('resellers.reseller_created'),
        description: t('common.object_created_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error creating reseller:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'organizations')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [RESELLERS_KEY] })
    queryCache.invalidateQueries({ key: [RESELLERS_TOTAL_KEY] })
  },
})

const {
  mutate: editResellerMutate,
  isLoading: editResellerLoading,
  reset: editResellerReset,
  error: editResellerError,
} = useMutation({
  mutation: (reseller: Reseller) => {
    return putReseller(reseller)
  },
  onSuccess(data, vars) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('resellers.reseller_saved'),
        description: t('common.object_saved_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error editing reseller:', error)
  },
  onSettled: () => queryCache.invalidateQueries({ key: [RESELLERS_KEY] }),
})

const name = ref('')
const nameRef = useTemplateRef<HTMLInputElement>('nameRef')
const description = ref('')
const descriptionRef = useTemplateRef<HTMLInputElement>('descriptionRef')
const vatNumber = ref('')
const vatNumberRef = useTemplateRef<HTMLInputElement>('vatNumberRef')
const validationIssues = ref<Record<string, string[]>>({})

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  name: nameRef,
  description: descriptionRef,
  'custom_data.vat': vatNumberRef,
}

const saving = computed(() => {
  return createResellerLoading.value || editResellerLoading.value
})

watch(
  () => isShown,
  () => {
    if (isShown) {
      clearErrors()
      focusElement(nameRef)

      if (currentReseller) {
        // editing reseller
        name.value = currentReseller.name
        description.value = currentReseller.description || ''
        vatNumber.value = currentReseller.custom_data?.vat || ''
      } else {
        // creating reseller, reset form to defaults
        name.value = ''
        description.value = ''
        vatNumber.value = ''
      }
    }
  },
)

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  createResellerReset()
  editResellerReset()
  validationIssues.value = {}
}

function validateCreate(reseller: CreateReseller): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(CreateResellerSchema, reseller) //// uncomment
  // const validation = { success: true } //// remove

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

      fieldRefs[firstErrorFieldName]?.value?.focus()
    }
    return false
  }
}

function validateEdit(reseller: Reseller): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(ResellerSchema, reseller)

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

      fieldRefs[firstErrorFieldName]?.value?.focus()
    }
    return false
  }
}

async function saveReseller() {
  clearErrors()

  const reseller = {
    name: name.value,
    description: description.value,
    custom_data: {
      vat: vatNumber.value,
    },
  }

  if (currentReseller?.id) {
    // editing reseller

    const resellerToEdit: Reseller = {
      ...reseller,
      id: currentReseller.id,
    }

    const isValidationOk = validateEdit(resellerToEdit)
    if (!isValidationOk) {
      return
    }
    editResellerMutate(resellerToEdit)
  } else {
    // creating reseller

    const resellerToCreate: CreateReseller = reseller

    const isValidationOk = validateCreate(resellerToCreate)
    if (!isValidationOk) {
      return
    }
    createResellerMutate(resellerToCreate)
  }
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="currentReseller ? $t('resellers.edit_reseller') : $t('resellers.create_reseller')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <!-- name -->
        <NeTextInput
          ref="nameRef"
          v-model.trim="name"
          :label="$t('organizations.name')"
          :invalid-message="validationIssues.name?.[0] ? $t(validationIssues.name[0]) : ''"
          :disabled="saving"
        />
        <!-- description -->
        <NeTextInput
          ref="descriptionRef"
          v-model.trim="description"
          :label="$t('organizations.description')"
          :invalid-message="
            validationIssues.description?.[0] ? $t(validationIssues.description[0]) : ''
          "
          :disabled="saving"
        />
        <!-- VAT number -->
        <NeTextInput
          ref="vatNumberRef"
          v-model.trim="vatNumber"
          :label="$t('organizations.vat_number')"
          :invalid-message="
            validationIssues['custom_data.vat']?.[0]
              ? $t(validationIssues['custom_data.vat'][0])
              : ''
          "
          :disabled="saving"
        />
        <!-- create reseller error notification -->
        <NeInlineNotification
          v-if="createResellerError?.message && !isValidationError(createResellerError)"
          kind="error"
          :title="t('resellers.cannot_create_reseller')"
          :description="createResellerError.message"
        />
        <!-- edit reseller error notification -->
        <NeInlineNotification
          v-if="editResellerError?.message && !isValidationError(editResellerError)"
          kind="error"
          :title="t('resellers.cannot_save_reseller')"
          :description="editResellerError.message"
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
          @click.prevent="saveReseller"
        >
          {{ currentReseller ? $t('resellers.save_reseller') : $t('resellers.create_reseller') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
