<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeSideDrawer, NeTextInput, NeInlineNotification } from '@nethesis/vue-components'
import OrganizationCombobox from '@/components/organizations/OrganizationCombobox.vue'
import { computed, ref } from 'vue'
import { useMutation, useQueryCache } from '@pinia/colada'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { getValidationIssues, isValidationError } from '@/lib/validation'
import {
  APPLICATIONS_KEY,
  assignOrganization,
  getDisplayName,
  type Application,
} from '@/lib/applications/applications'
import { type Organization } from '@/lib/organizations/organizations'
import type { AxiosError } from 'axios'

const { isShown = false, currentApplication = undefined } = defineProps<{
  isShown: boolean
  currentApplication: Application | undefined
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const organizationId = ref('')

// Company the backend infers for an unassigned application (from its
// hostnames, label, user domain or the system's owner): pre-selected, the
// user still has to save
const suggestion = computed(() =>
  currentApplication?.organization ? undefined : currentApplication?.suggested_organization,
)
const suggestionDescription = computed(() => {
  if (!suggestion.value) {
    return ''
  }
  const keyBySource: Record<string, string> = {
    display_name: 'applications.suggested_company_from_display_name',
    fqdn: 'applications.suggested_company_from_fqdn',
    user_domain: 'applications.suggested_company_from_user_domain',
    hosting_system: 'applications.suggested_company_from_system',
  }
  const key = keyBySource[suggestion.value.source] ?? 'applications.suggested_company_from_fqdn'
  return t(key, { matched: suggestion.value.matched, organization: suggestion.value.name })
})

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

const validationIssues = ref<Record<string, string[]>>({})

function onShow() {
  clearErrors()
  organizationId.value =
    currentApplication?.organization?.logto_id || suggestion.value?.logto_id || ''
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

  // name/type only feed the success notification: known for the current
  // company and for the suggested one, blank otherwise
  const known =
    organizationId.value === suggestion.value?.logto_id
      ? suggestion.value
      : currentApplication.organization
  const organization: Organization = {
    logto_id: organizationId.value,
    name: known?.name || '',
    description: '',
    type: known?.type || '',
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
    :close-aria-label="$t('shell.close_side_drawer')"
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
        <!-- suggested company, pre-selected below -->
        <NeInlineNotification
          v-if="suggestion"
          kind="info"
          :title="$t('applications.suggested_company')"
          :description="suggestionDescription"
        />
        <!-- organization -->
        <OrganizationCombobox
          ref="organizationIdRef"
          v-model="organizationId"
          :selected-organization="currentApplication?.organization ?? suggestion"
          :label="$t('organizations.organization')"
          :placeholder="$t('organizations.choose_organization')"
          :invalid-message="
            validationIssues.organization_id?.[0] ? $t(validationIssues.organization_id[0]) : ''
          "
          :disabled="assignOrganizationLoading"
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
