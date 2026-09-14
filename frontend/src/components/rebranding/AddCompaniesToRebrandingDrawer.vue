<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeInlineNotification,
  NeMultiselectCombobox,
  NeSideDrawer,
  type NeMultiselectComboboxOption,
} from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMutation, useQueryCache } from '@pinia/colada'
import type { AxiosError } from 'axios'
import { useAvailableRebrandingOrganizations } from '@/composables/useAvailableRebrandingOrganizations'
import { getValidationIssues, isValidationError } from '@/lib/validation'
import {
  REBRANDING_AVAILABLE_KEY,
  REBRANDING_ORGANIZATIONS_KEY,
  REBRANDING_SUMMARY_KEY,
} from '@/lib/rebranding/rebranding'
import { postEnableRebranding } from '@/lib/rebranding/rebrandingOrganizations'
import { useNotificationsStore } from '@/stores/notifications'

const { isShown = false } = defineProps<{
  isShown: boolean
}>()

const emit = defineEmits(['close'])

const { t } = useI18n()
const queryCache = useQueryCache()
const notificationsStore = useNotificationsStore()

const selectedCompanies = ref<NeMultiselectComboboxOption[]>([])
const validationIssues = ref<Record<string, string[]>>({})
const invalidCompanies = ref('')

// Nothing is fetched while the drawer is closed.
const { options, loading, onSearch, resetSearch } = useAvailableRebrandingOrganizations(
  () => isShown,
)

const {
  mutate: enableRebrandingMutate,
  isLoading: enableRebrandingLoading,
  reset: enableRebrandingReset,
  error: enableRebrandingError,
} = useMutation({
  mutation: (organizationIds: string[]) => postEnableRebranding(organizationIds),
  onSuccess(data) {
    const enabled = data.enabled

    // show success notification after the drawer closes
    setTimeout(() => {
      notificationsStore.createNotification({
        kind: 'success',
        title: t('rebranding.companies_enabled'),
        description: t('rebranding.companies_enabled_description', { count: enabled }, enabled),
      })
    }, 500)

    closeDrawer()
  },
  onError: (error) => {
    console.error('Error adding companies to rebranding:', error)
    validationIssues.value = getValidationIssues(error as AxiosError, 'rebranding')
  },
  onSettled: () => {
    queryCache.invalidateQueries({ key: [REBRANDING_ORGANIZATIONS_KEY] })
    queryCache.invalidateQueries({ key: [REBRANDING_SUMMARY_KEY] })
    // the companies just enabled must drop out of the picker
    queryCache.invalidateQueries({ key: [REBRANDING_AVAILABLE_KEY] })
  },
})

const companiesInvalidMessage = computed(() => {
  if (invalidCompanies.value) {
    return invalidCompanies.value
  }
  const issue = validationIssues.value.organization_ids?.[0]
  return issue ? t(issue) : ''
})

function onShow() {
  clearErrors()
  selectedCompanies.value = []
  resetSearch()
}

function closeDrawer() {
  emit('close')
}

function clearErrors() {
  enableRebrandingReset()
  validationIssues.value = {}
  invalidCompanies.value = ''
}

function enableRebranding() {
  clearErrors()

  if (!selectedCompanies.value.length) {
    invalidCompanies.value = t('rebranding.select_at_least_one_company')
    return
  }

  // The option id is the Logto id, which is what the endpoint resolves against.
  enableRebrandingMutate(selectedCompanies.value.map((company) => company.id))
}
</script>

<template>
  <NeSideDrawer
    :is-shown="isShown"
    :title="$t('rebranding.add_companies_to_rebranding')"
    :close-aria-label="$t('shell.close_side_drawer')"
    @show="onShow"
    @close="closeDrawer"
  >
    <form @submit.prevent>
      <div class="space-y-6">
        <NeMultiselectCombobox
          v-model="selectedCompanies"
          :options="options"
          :label="$t('rebranding.companies')"
          :placeholder="$t('rebranding.choose_company')"
          :helper-text="$t('rebranding.choose_company_helper')"
          :invalid-message="companiesInvalidMessage"
          :disabled="enableRebrandingLoading"
          external-filter
          :loading-options="loading"
          :no-results-label="$t('ne_combobox.no_results')"
          :no-options-label="$t('ne_combobox.no_options_label')"
          :limited-options-label="$t('ne_combobox.limited_options_label')"
          :user-input-label="$t('ne_combobox.user_input_label')"
          :optional-label="$t('common.optional')"
          @filter="onSearch"
        />
        <!-- enable rebranding error notification -->
        <NeInlineNotification
          v-if="enableRebrandingError?.message && !isValidationError(enableRebrandingError)"
          kind="error"
          :title="$t('rebranding.cannot_enable_rebranding')"
          :description="enableRebrandingError.message"
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
          :disabled="enableRebrandingLoading"
          :loading="enableRebrandingLoading"
          @click.prevent="enableRebranding"
        >
          {{ $t('common.enable') }}
        </NeButton>
      </div>
    </form>
  </NeSideDrawer>
</template>
