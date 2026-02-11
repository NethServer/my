<!--
  Copyright (C) 2026 Nethesis S.r.l.
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
import { getValidationIssues, isValidationError } from '@/lib/validation'
import {
  APPLICATIONS_KEY,
  assignOrganization,
  getDisplayName,
  type Application,
} from '@/lib/applications/applications'
import { getOrganizations, ORGANIZATIONS_KEY, type Organization } from '@/lib/organizations'
import { useLoginStore } from '@/stores/login'
import type { AxiosError } from 'axios'
import { organizationsQuery, useOrganizations } from '@/queries/organizations'

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
        description: t('applications.organization_assigned_description', {
          application: getDisplayName(vars),
          organization: vars.organization?.name,
        }),
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error assigning organization:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'applications')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [APPLICATIONS_KEY] })
  },
})

const { state: organizations } = useQuery({
  ...organizationsQuery,
  enabled: () => !!loginStore.jwtToken && isShown,
})

const validationIssues = ref<Record<string, string[]>>({})

const organizationOptions = computed(() => {
  if (!organizations.value.data) {
    return []
  }

  return organizations.value.data?.map((org) => ({
    id: org.logto_id,
    label: org.name,
    description: t(`organizations.${org.type}`),
  }))
})

function onShow() {
  clearErrors()
  organizationId.value = currentApplication?.organization?.logto_id || ''
}

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  assignOrganizationReset()
  validationIssues.value = {}
}

async function saveApplication() {
  clearErrors()

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
  assignOrganizationMutate(application)
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
          :label="$t('applications.application')"
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
