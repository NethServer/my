<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeSideDrawer, NeTextInput, NeInlineNotification } from '@nethesis/vue-components'
import OrganizationCombobox from '@/components/organizations/OrganizationCombobox.vue'
import { ref } from 'vue'
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
        <OrganizationCombobox
          ref="organizationIdRef"
          v-model="organizationId"
          :is-shown="isShown"
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
