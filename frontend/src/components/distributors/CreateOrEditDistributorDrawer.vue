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
  NeCombobox,
  type NeComboboxOption,
  getPreference,
} from '@nethesis/vue-components'
import { computed, ref, useTemplateRef, type ShallowRef } from 'vue'
import {
  CreateDistributorSchema,
  DISTRIBUTORS_KEY,
  DISTRIBUTORS_TOTAL_KEY,
  EditDistributorSchema,
  postDistributor,
  putDistributor,
  type CreateDistributor,
  type Distributor,
  type EditDistributor,
} from '@/lib/organizations/distributors'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { getValidationIssues, isValidationError } from '@/lib/validation'
import type { AxiosError } from 'axios'
import { getCommonLanguagesOptions } from '@/lib/locale'
import { getBrowserLocale } from '@/i18n'
import { useLoginStore } from '@/stores/login'

const { isShown = false, currentDistributor = undefined } = defineProps<{
  isShown: boolean
  currentDistributor: Distributor | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()
const loginStore = useLoginStore()

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
  mutation: (distributor: EditDistributor) => {
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
const vatNumber = ref('')
const vatNumberRef = useTemplateRef<HTMLInputElement>('vatNumberRef')
const address = ref('')
const addressRef = useTemplateRef<HTMLInputElement>('addressRef')
const city = ref('')
const cityRef = useTemplateRef<HTMLInputElement>('cityRef')
const mainContact = ref('')
const mainContactRef = useTemplateRef<HTMLInputElement>('mainContactRef')
const email = ref('')
const emailRef = useTemplateRef<HTMLInputElement>('emailRef')
const phone = ref('')
const phoneRef = useTemplateRef<HTMLInputElement>('phoneRef')
const language = ref('it')
const languageRef = useTemplateRef<HTMLInputElement>('languageRef')
const notes = ref('')
const notesRef = useTemplateRef<HTMLInputElement>('notesRef')
const validationIssues = ref<Record<string, string[]>>({})

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  name: nameRef,
  custom_data_vat: vatNumberRef,
  custom_data_address: addressRef,
  custom_data_city: cityRef,
  custom_data_main_contact: mainContactRef,
  custom_data_email: emailRef,
  custom_data_phone: phoneRef,
  custom_data_language: languageRef,
  custom_data_notes: notesRef,
}

const saving = computed(() => {
  return createDistributorLoading.value || editDistributorLoading.value
})

const languageOptions = computed((): NeComboboxOption[] => {
  if (loginStore.userInfo?.email && getPreference('locale', loginStore.userInfo.email)) {
    const locale = getPreference('locale', loginStore.userInfo.email)
    return getCommonLanguagesOptions(locale)
  } else {
    return getCommonLanguagesOptions(getBrowserLocale())
  }
})

function onShow() {
  clearErrors()
  focusElement(nameRef)

  if (currentDistributor) {
    // editing distributor
    name.value = currentDistributor.name
    vatNumber.value = currentDistributor.custom_data?.vat || ''
    address.value = currentDistributor.custom_data?.address || ''
    city.value = currentDistributor.custom_data?.city || ''
    mainContact.value = currentDistributor.custom_data?.main_contact || ''
    email.value = currentDistributor.custom_data?.email || ''
    phone.value = currentDistributor.custom_data?.phone || ''
    language.value = currentDistributor.custom_data?.language || ''
    notes.value = currentDistributor.custom_data?.notes || ''
  } else {
    // creating distributor, reset form to defaults
    name.value = ''
    vatNumber.value = ''
    address.value = ''
    city.value = ''
    mainContact.value = ''
    email.value = ''
    phone.value = ''
    language.value = 'it'
    notes.value = ''
  }
}

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

function validateEdit(distributor: EditDistributor): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(EditDistributorSchema, distributor)

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
    custom_data: {
      vat: vatNumber.value,
      address: address.value,
      city: city.value,
      main_contact: mainContact.value,
      email: email.value,
      phone: phone.value,
      language: language.value,
      notes: notes.value,
    },
  }

  if (currentDistributor?.logto_id) {
    // editing distributor

    const distributorToEdit: EditDistributor = {
      ...distributor,
      logto_id: currentDistributor.logto_id,
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
        <!-- address -->
        <NeTextInput
          ref="addressRef"
          v-model="address"
          @blur="address = address.trim()"
          :label="$t('organizations.address')"
          :invalid-message="
            validationIssues.custom_data_address?.[0]
              ? $t(validationIssues.custom_data_address[0])
              : ''
          "
          :disabled="saving"
          :optional="true"
          :optional-label="t('common.optional')"
        />
        <!-- city -->
        <NeTextInput
          ref="cityRef"
          v-model="city"
          @blur="city = city.trim()"
          :label="$t('organizations.city')"
          :invalid-message="
            validationIssues.custom_data_city?.[0] ? $t(validationIssues.custom_data_city[0]) : ''
          "
          :disabled="saving"
          :optional="true"
          :optional-label="t('common.optional')"
        />
        <!-- main contact -->
        <NeTextInput
          ref="mainContactRef"
          v-model="mainContact"
          @blur="mainContact = mainContact.trim()"
          :label="$t('organizations.main_contact')"
          :invalid-message="
            validationIssues.custom_data_main_contact?.[0]
              ? $t(validationIssues.custom_data_main_contact[0])
              : ''
          "
          :disabled="saving"
          :optional="true"
          :optional-label="t('common.optional')"
        />
        <!-- email -->
        <NeTextInput
          ref="emailRef"
          v-model="email"
          @blur="email = email.trim()"
          :label="$t('organizations.email')"
          :invalid-message="
            validationIssues.custom_data_email?.[0] ? $t(validationIssues.custom_data_email[0]) : ''
          "
          :disabled="saving"
          :optional="true"
          :optional-label="t('common.optional')"
        />
        <!-- phone -->
        <NeTextInput
          ref="phoneRef"
          v-model="phone"
          @blur="phone = phone.trim()"
          :label="$t('organizations.phone_number')"
          :invalid-message="
            validationIssues.custom_data_phone?.[0] ? $t(validationIssues.custom_data_phone[0]) : ''
          "
          :disabled="saving"
          :optional="true"
          :optional-label="t('common.optional')"
        />
        <!-- language -->
        <NeCombobox
          ref="languageRef"
          v-model="language"
          :options="languageOptions"
          :label="$t('organizations.language')"
          :placeholder="$t('ne_combobox.choose')"
          :invalid-message="
            validationIssues.custom_data_language?.[0]
              ? $t(validationIssues.custom_data_language[0])
              : ''
          "
          :disabled="saving"
          :optional="true"
          :optional-label="t('common.optional')"
          :no-results-label="$t('ne_combobox.no_results')"
          :limited-options-label="$t('ne_combobox.limited_options_label')"
          :no-options-label="$t('ne_combobox.no_options_label')"
          :selected-label="$t('ne_combobox.selected')"
          :user-input-label="$t('ne_combobox.user_input_label')"
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
