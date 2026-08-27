<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeCombobox,
  NeInlineNotification,
  NeSideDrawer,
  NeTextArea,
  NeTextInput,
  NeToggle,
  focusElement,
  type NeComboboxOption,
} from '@nethesis/vue-components'
import { useMutation, useQueryCache } from '@pinia/colada'
import type { AxiosError } from 'axios'
import * as v from 'valibot'
import { computed, ref, useTemplateRef, watch, type ShallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ADDON_APPLICATION_IDS,
  ADDONS_KEY,
  CreateAddonFormSchema,
  EditAddonFormSchema,
  composeAddonId,
  getAddonApplication,
  getAddonTechnicalName,
  getApplicationDisplayName,
  postAddon,
  putAddon,
  slugifyAddonName,
  type Addon,
  type AddonForm,
  type AddonProduct,
  type CreateAddon,
  type EditAddon,
} from '@/lib/addons/addons'
import { getBackendErrorMessage, getValidationIssues } from '@/lib/validation'
import { useNotificationsStore } from '@/stores/notifications'
import ProductTypeSelector from './ProductTypeSelector.vue'

const { isShown = false, currentAddon = undefined } = defineProps<{
  isShown: boolean
  currentAddon: Addon | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const {
  mutate: createAddonMutate,
  isLoading: createAddonLoading,
  reset: createAddonReset,
  error: createAddonError,
} = useMutation({
  mutation: (newAddon: CreateAddon) => {
    return postAddon(newAddon)
  },
  onSuccess(data, vars) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('addons.addon_created'),
        description: t('common.object_created_successfully', {
          name: vars.display_name,
        }),
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error creating add-on:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'addons')
  },
  onSettled: () => queryCache.invalidateQueries({ key: [ADDONS_KEY] }),
})

const {
  mutate: editAddonMutate,
  isLoading: editAddonLoading,
  reset: editAddonReset,
  error: editAddonError,
} = useMutation({
  mutation: (addon: EditAddon) => {
    return putAddon(addon)
  },
  onSuccess(data, vars) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('addons.addon_saved'),
        description: t('common.object_saved_successfully', {
          name: vars.display_name,
        }),
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error editing add-on:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'addons')
  },
  onSettled: () => queryCache.invalidateQueries({ key: [ADDONS_KEY] }),
})

const product = ref<AddonProduct | ''>('')
const application = ref('')
const applicationRef = useTemplateRef<HTMLInputElement>('applicationRef')
const technicalName = ref('')
const technicalNameEdited = ref(false)
const technicalNameRef = useTemplateRef<HTMLInputElement>('technicalNameRef')
const displayName = ref('')
const displayNameRef = useTemplateRef<HTMLInputElement>('displayNameRef')
const description = ref('')
const descriptionRef = useTemplateRef<HTMLInputElement>('descriptionRef')
const purchasable = ref(true)
const validationIssues = ref<Record<string, string[]>>({})

const fieldRefs: Record<string, Readonly<ShallowRef<HTMLInputElement | null>>> = {
  application: applicationRef,
  technical_name: technicalNameRef,
  display_name: displayNameRef,
  description: descriptionRef,
}

const saving = computed(() => {
  return createAddonLoading.value || editAddonLoading.value
})

// The product, the application and the technical name make up the add-on id,
// which the backend treats as immutable: editing only touches the display name
// and the description.
const isEditing = computed(() => !!currentAddon)

const applicationOptions = computed((): NeComboboxOption[] =>
  ADDON_APPLICATION_IDS.map((applicationId) => ({
    id: applicationId,
    label: getApplicationDisplayName(applicationId),
  })).sort((a, b) => a.label.localeCompare(b.label)),
)

const composedId = computed(() =>
  composeAddonId({
    product: product.value,
    application: application.value,
    technical_name: technicalName.value,
  }),
)

// One is usually the other in kebab-case, so the technical name follows the
// display name while the user has not written one. It stays a real field: the id
// it composes is immutable, and the feeds want terse ids (nsec-ha, not
// nsec-high-availability), so a deliberate value has to win.
watch(displayName, () => {
  if (!isEditing.value && !technicalNameEdited.value) {
    technicalName.value = slugifyAddonName(displayName.value)
  }
})

// A value the display name would not have produced can only have been typed:
// that is what stops the suggestion above, and clearing the field resumes it.
// NeTextInput emits update:modelValue only, so this comparison — rather than an
// @input listener — is what tells a manual edit from a suggested one.
watch(technicalName, (value) => {
  technicalNameEdited.value = !!value && value !== slugifyAddonName(displayName.value)
})

// A validation-code response with no field errors (e.g. the 409 raised when the
// id already exists) would otherwise be swallowed: the axios interceptor stays
// quiet for those and there is nothing to attach to a field.
function requestErrorMessage(error: Error | null) {
  if (!error || Object.keys(validationIssues.value).length) {
    return ''
  }
  return getBackendErrorMessage(error)
}

const createErrorMessage = computed(() => requestErrorMessage(createAddonError.value))
const editErrorMessage = computed(() => requestErrorMessage(editAddonError.value))

function onShow() {
  clearErrors()

  if (currentAddon) {
    // editing add-on
    product.value = (currentAddon.system_type as AddonProduct) || ''
    application.value = getAddonApplication(currentAddon)
    technicalName.value = getAddonTechnicalName(currentAddon)
    displayName.value = currentAddon.display_name
    description.value = currentAddon.description
    purchasable.value = currentAddon.purchasable
    technicalNameEdited.value = false
    focusElement(displayNameRef)
  } else {
    // creating add-on, reset form to defaults
    product.value = ''
    application.value = ''
    technicalName.value = ''
    technicalNameEdited.value = false
    displayName.value = ''
    description.value = ''
    // a new add-on is on sale unless it is deliberately withheld
    purchasable.value = true
  }
}

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  createAddonReset()
  editAddonReset()
  validationIssues.value = {}
}

function validate(schema: typeof CreateAddonFormSchema | typeof EditAddonFormSchema, form: object) {
  validationIssues.value = {}
  const validation = v.safeParse(schema, form)

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

      // focus the first field with error
      const firstErrorFieldName = Object.keys(validationIssues.value)[0]
      fieldRefs[firstErrorFieldName]?.value?.focus()
    }
    return false
  }
}

async function saveAddon() {
  clearErrors()

  if (currentAddon) {
    // editing add-on: the backend accepts the display fields only
    const form = {
      display_name: displayName.value,
      description: description.value,
    }

    if (!validate(EditAddonFormSchema, form)) {
      return
    }
    editAddonMutate({ ...form, id: currentAddon.id, purchasable: purchasable.value })
  } else {
    // creating add-on
    const form: AddonForm = {
      product: product.value,
      application: application.value,
      technical_name: technicalName.value,
      display_name: displayName.value,
      description: description.value,
    }

    if (!validate(CreateAddonFormSchema, form)) {
      return
    }

    createAddonMutate({
      id: composedId.value,
      display_name: form.display_name,
      description: form.description,
      // Both derived from the product: NethSecurity add-ons are firewall-wide
      // services, NethServer add-ons are modules of a single application
      // instance of a cluster.
      kind: form.product === 'ns8' ? 'module' : 'service',
      system_type: form.product as AddonProduct,
      scoped: form.product === 'ns8',
      // the application the module belongs to, so the backend does not have to
      // guess it from the id prefix
      applies_to: form.application,
      purchasable: purchasable.value,
    })
  }
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="currentAddon ? $t('addons.edit_addon') : $t('addons.create_addon')"
    :close-aria-label="$t('common.close')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <!-- product type -->
        <ProductTypeSelector
          v-model="product"
          :label="$t('addons.product_type')"
          :disabled="saving || isEditing"
          :invalid-message="validationIssues.product?.[0] ? $t(validationIssues.product[0]) : ''"
        />
        <!-- application (NethServer add-ons only) -->
        <NeCombobox
          v-if="product === 'ns8'"
          ref="applicationRef"
          v-model="application"
          :options="applicationOptions"
          :label="$t('addons.application')"
          :placeholder="$t('addons.choose_application')"
          :invalid-message="
            validationIssues.application?.[0] ? $t(validationIssues.application[0]) : ''
          "
          :disabled="saving || isEditing"
          :no-results-label="$t('ne_combobox.no_results')"
          :limited-options-label="$t('ne_combobox.limited_options_label')"
          :no-options-label="$t('ne_combobox.no_options_label')"
          :selected-label="$t('ne_combobox.selected')"
          :user-input-label="$t('ne_combobox.user_input_label')"
          :optional-label="$t('common.optional')"
        />
        <!-- display name -->
        <NeTextInput
          ref="displayNameRef"
          v-model="displayName"
          @blur="displayName = displayName.trim()"
          :label="$t('addons.display_name')"
          :placeholder="$t('common.eg_value', { value: 'Advanced Threat Shield' })"
          :helper-text="$t('addons.display_name_helper')"
          :invalid-message="
            validationIssues.display_name?.[0] ? $t(validationIssues.display_name[0]) : ''
          "
          :disabled="saving"
        />
        <!-- technical name -->
        <NeTextInput
          ref="technicalNameRef"
          v-model="technicalName"
          @blur="technicalName = technicalName.trim()"
          :label="$t('addons.technical_name')"
          :placeholder="
            $t('common.eg_value', {
              value: product === 'ns8' ? 'blocklist' : 'advanced-threat-shield',
            })
          "
          :helper-text="
            composedId
              ? $t('addons.technical_name_helper_with_id', { id: composedId })
              : $t('addons.technical_name_helper')
          "
          :invalid-message="
            validationIssues.technical_name?.[0] ? $t(validationIssues.technical_name[0]) : ''
          "
          :disabled="saving || isEditing"
        />
        <!-- description -->
        <NeTextArea
          ref="descriptionRef"
          v-model="description"
          @blur="description = description.trim()"
          :label="$t('addons.description')"
          :invalid-message="
            validationIssues.description?.[0] ? $t(validationIssues.description[0]) : ''
          "
          :disabled="saving"
          :optional="true"
          :optional-label="t('common.optional')"
        />
        <!-- on sale: withholding it keeps the add-on everywhere it is granted
             and only takes away the buy action -->
        <NeToggle
          v-model="purchasable"
          :top-label="$t('addons.on_sale_label')"
          :label="purchasable ? $t('addons.on_sale') : $t('addons.not_on_sale')"
          :disabled="saving"
        />
        <p class="text-tertiary-neutral text-sm">
          {{ $t('addons.on_sale_helper') }}
        </p>
        <!-- create add-on error notification -->
        <NeInlineNotification
          v-if="createErrorMessage"
          kind="error"
          :title="t('addons.cannot_create_addon')"
          :description="createErrorMessage"
        />
        <!-- edit add-on error notification -->
        <NeInlineNotification
          v-if="editErrorMessage"
          kind="error"
          :title="t('addons.cannot_save_addon')"
          :description="editErrorMessage"
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
          @click.prevent="saveAddon"
        >
          {{ currentAddon ? $t('addons.save_addon') : $t('addons.create_addon') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
