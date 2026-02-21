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
  CreateResellerSchema,
  RESELLERS_KEY,
  RESELLERS_TOTAL_KEY,
  EditResellerSchema,
  postReseller,
  putReseller,
  type CreateReseller,
  type Reseller,
  type EditReseller,
} from '@/lib/organizations/resellers'
import * as v from 'valibot'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { getValidationIssues, isValidationError } from '@/lib/validation'
import type { AxiosError } from 'axios'
import { getCommonLanguagesOptions } from '@/lib/locale'
import { getBrowserLocale } from '@/i18n'
import { useLoginStore } from '@/stores/login'

const { isShown = false, currentReseller = undefined } = defineProps<{
  isShown: boolean
  currentReseller: Reseller | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()
const loginStore = useLoginStore()

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
  mutation: (reseller: EditReseller) => {
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
  return createResellerLoading.value || editResellerLoading.value
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

  if (currentReseller) {
    // editing reseller
    name.value = currentReseller.name
    vatNumber.value = currentReseller.custom_data?.vat || ''
    address.value = currentReseller.custom_data?.address || ''
    city.value = currentReseller.custom_data?.city || ''
    mainContact.value = currentReseller.custom_data?.main_contact || ''
    email.value = currentReseller.custom_data?.email || ''
    phone.value = currentReseller.custom_data?.phone || ''
    language.value = currentReseller.custom_data?.language || ''
    notes.value = currentReseller.custom_data?.notes || ''
  } else {
    // creating reseller, reset form to defaults
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
  createResellerReset()
  editResellerReset()
  validationIssues.value = {}
}

function validateCreate(reseller: CreateReseller): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(CreateResellerSchema, reseller)

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

function validateEdit(reseller: EditReseller): boolean {
  validationIssues.value = {}
  const validation = v.safeParse(EditResellerSchema, reseller)

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

async function saveReseller() {
  clearErrors()

  const reseller = {
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

  if (currentReseller?.logto_id) {
    // editing reseller

    const resellerToEdit: EditReseller = {
      ...reseller,
      logto_id: currentReseller.logto_id,
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
          :label="$t('common.notes')"
          :disabled="saving"
          :invalid-message="validationIssues.notes?.[0] ? $t(validationIssues.notes[0]) : ''"
          :optional="true"
          :optional-label="t('common.optional')"
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
