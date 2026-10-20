<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeMultiselectCombobox, NeRadioSelection } from '@nethesis/vue-components'
import type { NeMultiselectComboboxOption, RadioOption } from '@nethesis/vue-components'
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { useOrganizationFilter } from '@/composables/useOrganizationFilter'
import { useSystemFilter } from '@/composables/useSystemFilter'
import type { AlertScopeNames, RecipientScope } from '@/lib/alerts'
import { isUserCustomer } from '@/lib/organizations/organizations'

const props = defineProps<{
  scopeNames?: AlertScopeNames
  invalidMessage?: string
}>()

const specific = defineModel<boolean>('specific', { required: true })
const scope = defineModel<RecipientScope | undefined>('scope', { required: true })

const { t } = useI18n()

// A customer manages no other company and picking its own one means every
// system, so it narrows by system only.
const canPickOrganizations = !isUserCustomer()

const modeOptions = computed<RadioOption[]>(() => [
  { id: 'all', label: t('alerts.scope_all_systems') },
  {
    id: 'specific',
    label: t(canPickOrganizations ? 'alerts.scope_specific' : 'alerts.scope_specific_systems'),
  },
])

const mode = computed({
  get: () => (specific.value ? 'specific' : 'all'),
  set: (value: string) => {
    specific.value = value === 'specific'
  },
})

const {
  organizations,
  loading: organizationsLoading,
  onSearch: onOrganizationSearch,
} = useOrganizationFilter(() => specific.value && canPickOrganizations)
const { systems, loading: systemsLoading, onSearch: onSystemSearch } = useSystemFilter('id')

// Labels of the selected ids: seeded from the names the API resolved for the
// stored scope, extended with whatever the user picks from a search page.
const labels = reactive<Record<string, NeMultiselectComboboxOption>>({})

const organizationOptions = computed<NeMultiselectComboboxOption[]>(() =>
  organizations.value.map((org) => ({
    id: org.logto_id,
    label: org.name,
    description: t(`organizations.${org.type}`),
  })),
)

const systemOptions = computed<NeMultiselectComboboxOption[]>(() =>
  systems.value.map((sys) => ({
    id: sys.id,
    label: sys.name,
    description: sys.organization?.name,
  })),
)

function organizationOption(id: string): NeMultiselectComboboxOption {
  const known = labels[id] ?? organizationOptions.value.find((o) => o.id === id)
  if (known) return known
  const named = props.scopeNames?.organizations?.[id]
  return named
    ? { id, label: named.name, description: t(`organizations.${named.type}`) }
    : { id, label: t('alerts.scope_unknown_organization') }
}

function systemOption(id: string): NeMultiselectComboboxOption {
  const known = labels[id] ?? systemOptions.value.find((o) => o.id === id)
  if (known) return known
  const named = props.scopeNames?.systems?.[id]
  return named
    ? { id, label: named.name, description: named.organization_name }
    : { id, label: t('alerts.scope_unknown_system') }
}

function remember(options: NeMultiselectComboboxOption[]) {
  for (const option of options) {
    labels[option.id] = option
  }
}

const selectedOrganizations = computed<NeMultiselectComboboxOption[]>({
  get: () => (scope.value?.organization_ids ?? []).map(organizationOption),
  set: (options) => {
    remember(options)
    scope.value = { ...scope.value, organization_ids: options.map((o) => o.id) }
  },
})

const selectedSystems = computed<NeMultiselectComboboxOption[]>({
  get: () => (scope.value?.system_ids ?? []).map(systemOption),
  set: (options) => {
    remember(options)
    scope.value = { ...scope.value, system_ids: options.map((o) => o.id) }
  },
})
</script>

<template>
  <div class="space-y-4">
    <NeRadioSelection v-model="mode" :options="modeOptions" :label="t('alerts.scope')" />
    <template v-if="specific">
      <NeMultiselectCombobox
        v-if="canPickOrganizations"
        v-model="selectedOrganizations"
        :options="organizationOptions"
        :label="t('alerts.scope_organizations')"
        :placeholder="t('organizations.choose_organizations')"
        :helper-text="t('alerts.scope_organizations_helper')"
        :loading-options="organizationsLoading"
        :no-results-label="t('ne_combobox.no_results')"
        :no-options-label="t('organizations.no_organizations')"
        :limited-options-label="t('ne_combobox.limited_options_label')"
        :user-input-label="t('ne_combobox.user_input_label')"
        :optional-label="t('common.optional')"
        external-filter
        @filter="onOrganizationSearch"
      />
      <NeMultiselectCombobox
        v-model="selectedSystems"
        :options="systemOptions"
        :label="t('alerts.scope_systems')"
        :placeholder="t('systems.choose_systems')"
        :invalid-message="props.invalidMessage"
        :loading-options="systemsLoading"
        :no-results-label="t('ne_combobox.no_results')"
        :no-options-label="t('alerts.scope_no_systems')"
        :limited-options-label="t('ne_combobox.limited_options_label')"
        :user-input-label="t('ne_combobox.user_input_label')"
        :optional-label="t('common.optional')"
        external-filter
        @filter="onSystemSearch"
      />
    </template>
  </div>
</template>
