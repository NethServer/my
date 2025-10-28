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
  CreateDistributorSchema,
  DISTRIBUTORS_KEY,
  DISTRIBUTORS_TOTAL_KEY,
  DistributorSchema,
  postDistributor,
  putDistributor,
  type CreateDistributor,
  type Distributor,
} from '@/lib/distributors'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { getValidationIssues, isValidationError } from '@/lib/validation'
import type { AxiosError } from 'axios'

const { isShown = false, currentDistributor = undefined } = defineProps<{
  isShown: boolean
  currentDistributor: Distributor | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const {
  mutate: createDistributorMutate,
  isLoading: createDistributorLoading,
  reset: createDistributorReset,
  error: createDistributorError,
} = useMutation({
  mutation: (newDistributor: CreateDistributor) => {
    return postDistributor(newDistributor)
  },
  onSuccess(data, vars) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('distributors.distributor_created'),
        description: t('common.object_created_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error creating distributor:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'organizations')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_KEY] })
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_TOTAL_KEY] })
  },
})

const {
  mutate: editDistributorMutate,
  isLoading: editDistributorLoading,
  reset: editDistributorReset,
  error: editDistributorError,
} = useMutation({
  mutation: (distributor: Distributor) => {
    return putDistributor(distributor)
  },
  onSuccess(data, vars) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('distributors.distributor_saved'),
        description: t('common.object_saved_successfully', {
          name: vars.name,
        }),
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error editing distributor:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [DISTRIBUTORS_KEY] })
  },
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
  custom_data_vat: vatNumberRef,
}

const saving = computed(() => {
  return createDistributorLoading.value || editDistributorLoading.value
})

watch(
  () => isShown,
  () => {
    if (isShown) {
      clearErrors()
      focusElement(nameRef)

      if (currentDistributor) {
        // editing distributor
        name.value = currentDistributor.name
        description.value = currentDistributor.description || ''
        vatNumber.value = currentDistributor.custom_data?.vat || ''
      } else {
        // creating distributor, reset form to defaults
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
  createDistributorReset()
  editDistributorReset()
  validationIssues.value = {}
}

function validateCreate(distributor: CreateDistributor): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(CreateDistributorSchema, distributor)

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

function validateEdit(distributor: Distributor): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(DistributorSchema, distributor)

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

async function saveDistributor() {
  clearErrors()

  const distributor = {
    name: name.value,
    description: description.value,
    custom_data: {
      vat: vatNumber.value,
    },
  }

  if (currentDistributor?.id) {
    // editing distributor

    const distributorToEdit: Distributor = {
      ...distributor,
      id: currentDistributor.id,
    }

    const isValidationOk = validateEdit(distributorToEdit)
    if (!isValidationOk) {
      return
    }
    editDistributorMutate(distributorToEdit)
  } else {
    // creating distributor

    const distributorToCreate: CreateDistributor = distributor
    const isValidationOk = validateCreate(distributorToCreate)
    if (!isValidationOk) {
      return
    }
    createDistributorMutate(distributorToCreate)
  }
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="
      currentDistributor
        ? $t('distributors.edit_distributor')
        : $t('distributors.create_distributor')
    "
    :close-aria-label="$t('common.shell.close_side_drawer')"
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
        <!-- create distributor error notification -->
        <NeInlineNotification
          v-if="createDistributorError?.message && !isValidationError(createDistributorError)"
          kind="error"
          :title="t('distributors.cannot_create_distributor')"
          :description="createDistributorError.message"
        />
        <!-- edit distributor error notification -->
        <NeInlineNotification
          v-if="editDistributorError?.message && !isValidationError(editDistributorError)"
          kind="error"
          :title="t('distributors.cannot_save_distributor')"
          :description="editDistributorError.message"
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
          @click.prevent="saveDistributor"
        >
          {{
            currentDistributor
              ? $t('distributors.save_distributor')
              : $t('distributors.create_distributor')
          }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
