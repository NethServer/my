<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeDropdownFilterV2, type NeDropdownFilterV2Option } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'
import { WITH_COMMENTS_FILTER_ID, WITHOUT_COMMENTS_FILTER_ID } from '@/lib/alerts'

const { modelValue } = defineProps<{
  modelValue: NeDropdownFilterV2Option[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: NeDropdownFilterV2Option[]]
}>()

const { t } = useI18n()

const options = computed<NeDropdownFilterV2Option[]>(() => [
  { id: WITH_COMMENTS_FILTER_ID, label: t('alerts.with_comments') },
  { id: WITHOUT_COMMENTS_FILTER_ID, label: t('alerts.without_comments') },
])
</script>

<template>
  <NeDropdownFilterV2
    :model-value="modelValue"
    kind="checkbox"
    :label="t('alerts.comments')"
    :options="options"
    :clear-filter-label="t('ne_dropdown_filter.clear_selection')"
    :open-menu-aria-label="t('ne_dropdown_filter.open_filter')"
    :no-options-label="t('ne_dropdown_filter.no_options')"
    :more-options-hidden-label="t('ne_dropdown_filter.more_options_hidden')"
    :clear-search-label="t('ne_dropdown_filter.clear_search')"
    :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
    @update:model-value="(val) => emit('update:modelValue', val ?? [])"
  />
</template>
