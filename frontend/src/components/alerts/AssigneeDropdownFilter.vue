<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeDropdownFilterV2, type NeDropdownFilterV2Option } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'
import { useAssigneeFilter } from '@/composables/useAssigneeFilter'
import { UNASSIGNED_FILTER_ID } from '@/lib/alerts'
import { OPTIONS_PAGE_SIZE } from '@/lib/common'
import { useLoginStore } from '@/stores/login'

const { modelValue, label } = defineProps<{
  modelValue: NeDropdownFilterV2Option[]
  label?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: NeDropdownFilterV2Option[]]
}>()

const { t } = useI18n()
const loginStore = useLoginStore()
const computedLabel = label ?? t('alerts.assigned_to')
const { options, loading, onSearch, currentSearch } = useAssigneeFilter()

const unassignedLabel = computed(() => t('alerts.unassigned'))

// The current user's assignee id (Logto id when present, else the local id).
const meId = computed(() => loginStore.userInfo?.logto_id ?? loginStore.userInfo?.id ?? '')
const meLabel = computed(() => t('alerts.assignee_me', { name: loginStore.userDisplayName }))

// Synthetic options at the top, matching the search like the fetched ones:
//   1. "Unassigned" (backend sentinel "none")
//   2. "<my name> (me)"
// The current user is removed from the fetched list to avoid a duplicate.
const finalOptions = computed<NeDropdownFilterV2Option[]>(() => {
  const search = currentSearch.value.toLowerCase()
  const result: NeDropdownFilterV2Option[] = []

  if (!search || unassignedLabel.value.toLowerCase().includes(search)) {
    result.push({ id: UNASSIGNED_FILTER_ID, label: unassignedLabel.value })
  }
  if (meId.value && (!search || meLabel.value.toLowerCase().includes(search))) {
    result.push({ id: meId.value, label: meLabel.value })
  }
  result.push(...options.value.filter((o) => o.id !== meId.value))

  return result
})
</script>

<template>
  <NeDropdownFilterV2
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
    :options-filter-placeholder="t('ne_dropdown_filter.options_filter_placeholder')"
    :max-options-shown="OPTIONS_PAGE_SIZE"
    @search="onSearch"
    @update:model-value="(val) => emit('update:modelValue', val ?? [])"
  />
</template>
