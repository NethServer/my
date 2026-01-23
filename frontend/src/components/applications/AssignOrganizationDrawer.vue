<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeSideDrawer,
  NeTextInput,
  NeInlineNotification,
  NeCombobox,
} from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { isValidationError } from '@/lib/validation'
import {
  APPLICATIONS_KEY,
  assignOrganization,
  getDisplayName,
  type Application,
  type Organization,
} from '@/lib/applications'
import { getOrganizations, ORGANIZATIONS_KEY } from '@/lib/organizations'
import { useLoginStore } from '@/stores/login'
import { get } from 'lodash'

//// review (search "distributor")

const { isShown = false, currentApplication = undefined } = defineProps<{
  isShown: boolean
  currentApplication: Application | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const loginStore = useLoginStore()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const organizationId = ref('')

const {
  mutate: assignOrganizationMutate,
  isLoading: assignOrganizationLoading,
  reset: assignOrganizationReset,
  error: assignOrganizationError,
} = useMutation({
  mutation: async (application: Application) => {
    if (application.organization?.logto_id) {
      return assignOrganization(application.organization?.logto_id, application.id)
    }
    throw new Error('Organization ID is required')
  },
  onSuccess(data, vars) {
    // show success notification after drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('applications.organization_assigned'),
        description: t('applications.organization_assigned_description', {}), //// fix description with vars
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error assigning organization:', error)
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [APPLICATIONS_KEY] })
  },
})

const { state: organizations } = useQuery({
  key: [ORGANIZATIONS_KEY],
  enabled: () => !!loginStore.jwtToken && isShown,
  query: getOrganizations,
})

// const name = ref('') ////
// const nameRef = useTemplateRef<HTMLInputElement>('nameRef')
// const description = ref('')
// const descriptionRef = useTemplateRef<HTMLInputElement>('descriptionRef')
// const vatNumber = ref('')
// const vatNumberRef = useTemplateRef<HTMLInputElement>('vatNumberRef')
const validationIssues = ref<Record<string, string[]>>({})

const organizationOptions = computed(() => {
  if (!organizations.value.data) {
    return []
  }

  return organizations.value.data?.map((org) => ({
    id: org.id,
    label: org.name,
    description: t(`organizations.${org.type}`),
  }))
})

function onShow() {
  clearErrors()

  if (currentApplication) {
    // editing application
    organizationId.value = currentApplication.organization?.logto_id || ''
  }
}

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  assignOrganizationReset()
  validationIssues.value = {}
}

////
// function validateEdit(application: Application): boolean {
//   validationIssues.value = {}
//   const validation = v.safeParse(ApplicationSchema, application)

//   if (validation.success) {
//     // no validation issues
//     return true
//   } else {
//     const flattenedIssues = v.flatten(validation.issues)

//     if (flattenedIssues.nested) {
//       const issues: Record<string, string[]> = {}

//       for (const key in flattenedIssues.nested) {
//         // replace dots with underscores for i18n key
//         const newKey = key.replace(/\./g, '_')
//         issues[newKey] = flattenedIssues.nested[key] ?? []
//       }
//       validationIssues.value = issues

//       console.debug('frontend validation issues', validationIssues.value)

//       // focus the first field with error

//       const firstErrorFieldName = Object.keys(validationIssues.value)[0]
//       fieldRefs[firstErrorFieldName]?.value?.focus()
//     }
//     return false
//   }
// }

async function saveApplication() {
  clearErrors()

  // const application = { ////
  //   name: name.value,
  //   description: description.value,
  //   custom_data: {
  //     vat: vatNumber.value,
  //   },
  // }

  // if (currentApplication?.id) { ////
  // editing application

  if (!currentApplication) {
    return
  }

  const organization: Organization = {
    logto_id: organizationId.value,
    name: currentApplication.organization?.name || '',
    description: currentApplication.organization?.description || '',
    type: currentApplication.organization?.type || '',
  }

  const application: Application = {
    ...currentApplication,
    organization,
  }

  ////
  // const isValidationOk = validateEdit(distributorToEdit)
  // if (!isValidationOk) {
  //   return
  // }

  assignOrganizationMutate(application)

  // } else { ////
  //   // creating application

  //   const distributorToCreate: CreateApplication = application
  //   const isValidationOk = validateCreate(distributorToCreate)
  //   if (!isValidationOk) {
  //     return
  //   }
  //   createApplicationMutate(distributorToCreate)
  // }
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="$t('applications.assign_organization')"
    :close-aria-label="$t('common.shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <!-- name -->
        <NeTextInput
          :value="currentApplication ? getDisplayName(currentApplication) : ''"
          :label="$t('applications.application_name')"
          readonly
        />
        <!-- type -->
        <NeTextInput
          :value="currentApplication?.instance_of"
          :label="$t('applications.type')"
          readonly
        />
        <!-- organization -->
        <NeCombobox
          ref="organizationIdRef"
          v-model="organizationId"
          :options="organizationOptions"
          :label="$t('organizations.organization')"
          :placeholder="
            organizations.status === 'pending'
              ? $t('common.loading')
              : $t('organizations.choose_organization')
          "
          :invalid-message="
            validationIssues.organization_id?.[0] ? $t(validationIssues.organization_id[0]) : ''
          "
          :disabled="organizations.status === 'pending' || assignOrganizationLoading"
          :no-results-label="$t('ne_combobox.no_results')"
          :limited-options-label="$t('ne_combobox.limited_options_label')"
          :no-options-label="$t('organizations.no_organizations')"
          :selected-label="$t('ne_combobox.selected')"
          :user-input-label="$t('ne_combobox.user_input_label')"
          :optional-label="$t('common.optional')"
        />
        <!-- assign organization error notification -->
        <NeInlineNotification
          v-if="assignOrganizationError?.message && !isValidationError(assignOrganizationError)"
          kind="error"
          :title="t('applications.cannot_assign_organization')"
          :description="assignOrganizationError.message"
        />
      </div>
      <!-- footer -->
      <hr class="my-8" />
      <div class="flex justify-end">
        <NeButton kind="tertiary" size="lg" class="mr-3" @click.prevent="closeDrawer">
          {{ $t('common.cancel') }}
        </NeButton>
        <NeButton
          type="submit"
          kind="primary"
          size="lg"
          :disabled="assignOrganizationLoading"
          :loading="assignOrganizationLoading"
          @click.prevent="saveApplication"
        >
          {{ $t('common.save') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
