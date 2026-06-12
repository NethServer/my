<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeDropdownFilter } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useOrganizationFilter } from '@/composables/useOrganizationFilter'

const { modelValue, label } = defineProps<{
  modelValue: string[]
  label?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const { t } = useI18n()
const computedLabel = label ?? t('organizations.organization')
const { options, loading, onSearch } = useOrganizationFilter()
</script>

<template>
  <NeDropdownFilter
    :model-value="modelValue"
    kind="checkbox"
    :label="computedLabel"
    :options="options"
    show-options-filter
    external-filter
    :loading-options="loading"
    :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
    :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
    :no-options-label="t('ne_dropdown_filter.no_options')"
    :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
    :clear-search-label="t('ne_dropdown_filter.clear_search')"
    :max-options-shown="50"
    @search="onSearch"
    @update:model-value="(val) => emit('update:modelValue', val ?? [])"
  />
</template>
