<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeDropdownFilter } from '@nethesis/vue-components'
import type { FilterOption } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useOrganizationFilter } from '@/composables/useOrganizationFilter'
import { computed } from 'vue'
import { COMBOBOX_PAGE_SIZE } from '@/lib/common'

const {
  modelValue,
  label,
  showNoCompanyOption = false,
} = defineProps<{
  modelValue: string[]
  label?: string
  showNoCompanyOption?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const { t } = useI18n()
const computedLabel = label ?? t('organizations.organization')
const { options, loading, onSearch, currentSearch } = useOrganizationFilter()

const noCompanyLabel = computed(() => t('organizations.no_company'))

const finalOptions = computed<FilterOption[]>(() => {
  if (!showNoCompanyOption) return options.value
  const search = currentSearch.value.toLowerCase()
  const matches = !search || noCompanyLabel.value.toLowerCase().includes(search)
  if (matches) {
    return [{ id: 'no_org', label: noCompanyLabel.value }, ...options.value]
  }
  return options.value
})
</script>

<template>
  <NeDropdownFilter
    :model-value="modelValue"
    kind="checkbox"
    :label="computedLabel"
    :options="finalOptions"
    show-options-filter
    external-filter
    :loading-options="loading"
    :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
    :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
    :no-options-label="t('ne_dropdown_filter.no_options')"
    :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
    :clear-search-label="t('ne_dropdown_filter.clear_search')"
    :max-options-shown="COMBOBOX_PAGE_SIZE"
    @search="onSearch"
    @update:model-value="(val) => emit('update:modelValue', val ?? [])"
  />
</template>
