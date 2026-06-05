<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeDropdownFilter } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { useSystemFilter } from '@/composables/useSystemFilter'

const {
  modelValue,
  label,
  idField = 'system_key',
} = defineProps<{
  modelValue: string[]
  label?: string
  idField?: 'system_key' | 'id'
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const { t } = useI18n()
const computedLabel = label ?? t('systems.system')
const { options, loading, onSearch } = useSystemFilter(idField)
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
    :no-options-label="t('systems.no_systems')"
    :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
    :clear-search-label="t('ne_dropdown_filter.clear_search')"
    :max-options-shown="50"
    @search="onSearch"
    @update:model-value="(val) => emit('update:modelValue', val ?? [])"
  />
</template>
