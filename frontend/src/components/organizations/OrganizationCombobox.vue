<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCombobox } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { computed, ref } from 'vue'
import { useOrganizationFilter } from '@/composables/useOrganizationFilter'

const props = withDefaults(
  defineProps<{
    modelValue: string
    // Gates the org search so it doesn't fire while the host drawer is closed.
    isShown?: boolean
    label: string
    disabled?: boolean
    invalidMessage?: string
    helperText?: string
    placeholder?: string
    optional?: boolean
    // Restrict the options to these organization types (e.g. ['distributor']).
    allowedTypes?: string[]
    // Exclude these organization ids from the options (e.g. the caller's own org).
    excludeOrganizationIds?: string[]
  }>(),
  {
    isShown: true,
    optional: false,
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { t } = useI18n()

const { organizations, loading, onSearch, currentSearch } = useOrganizationFilter(
  () => props.isShown,
)

const organizationOptions = computed(() => {
  let orgs = organizations.value
  if (props.allowedTypes && props.allowedTypes.length > 0) {
    orgs = orgs.filter((org) => props.allowedTypes!.includes(org.type))
  }
  if (props.excludeOrganizationIds && props.excludeOrganizationIds.length > 0) {
    orgs = orgs.filter((org) => !props.excludeOrganizationIds!.includes(org.logto_id))
  }
  return orgs.map((org) => ({
    id: org.logto_id,
    label: org.name,
    description: t(`organizations.${org.type}`),
  }))
})

const isLoading = loading

const isInitiallyLoading = computed(
  () => loading.value && organizations.value.length === 0 && !currentSearch.value,
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
    :optional="props.optional"
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
    @filter="onSearch"
  >
    <!-- forward the tooltip slot only when the caller provides one, so other
      consumers don't get an empty tooltip icon -->
    <template v-if="$slots.tooltip" #tooltip>
      <slot name="tooltip" />
    </template>
  </NeCombobox>
</template>
