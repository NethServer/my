<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCombobox } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { computed, ref, watch } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import {
  ORGANIZATIONS_SEARCH_KEY,
  searchOrganizations,
} from '@/lib/organizations/searchOrganizations'

const props = withDefaults(
  defineProps<{
    modelValue: string
    isShown?: boolean
    label: string
    disabled?: boolean
    invalidMessage?: string
    helperText?: string
    placeholder?: string
  }>(),
  {
    isShown: true,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { t } = useI18n()
const loginStore = useLoginStore()

const orgSearchInput = ref('')
const debouncedOrgSearch = ref('')

watch(
  () => orgSearchInput.value,
  useDebounceFn(() => {
    debouncedOrgSearch.value = orgSearchInput.value
  }, 300),
)

const { state: organizations } = useQuery({
  key: () => [ORGANIZATIONS_SEARCH_KEY, debouncedOrgSearch.value],
  enabled: () => !!loginStore.jwtToken && props.isShown,
  query: () => searchOrganizations(debouncedOrgSearch.value),
})

const organizationOptions = computed(() => {
  if (!organizations.value.data) return []
  return organizations.value.data.map((org) => ({
    id: org.logto_id,
    label: org.name,
    description: t(`organizations.${org.type}`),
  }))
})

const isLoading = computed(() => organizations.value.status === 'pending')

const isInitiallyLoading = computed(
  () =>
    organizations.value.status === 'pending' && !organizations.value.data && !orgSearchInput.value,
)

const computedPlaceholder = computed(() =>
  isLoading.value
    ? t('common.loading')
    : (props.placeholder ?? t('organizations.choose_organization')),
)

const comboboxRef = ref()

defineExpose({
  focus: () => comboboxRef.value?.focus?.(),
})
</script>

<template>
  <NeCombobox
    ref="comboboxRef"
    :model-value="props.modelValue"
    :options="organizationOptions"
    :label="props.label"
    :placeholder="computedPlaceholder"
    :invalid-message="props.invalidMessage"
    :disabled="isInitiallyLoading || props.disabled"
    :no-results-label="t('ne_combobox.no_results')"
    :limited-options-label="t('ne_combobox.limited_options_label')"
    :no-options-label="t('organizations.no_organizations')"
    :selected-label="t('ne_combobox.selected')"
    :user-input-label="t('ne_combobox.user_input_label')"
    :optional-label="t('common.optional')"
    :loading-options="isLoading"
    :helper-text="props.helperText"
    external-filter
    @update:model-value="emit('update:modelValue', $event)"
    @filter="orgSearchInput = $event"
  />
</template>
