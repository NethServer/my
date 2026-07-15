<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { NeTooltip } from '@nethesis/vue-components'
import OrganizationCombobox from '@/components/organizations/OrganizationCombobox.vue'
import { useLoginStore } from '@/stores/login'

const props = withDefaults(
  defineProps<{
    modelValue: string
    // Lowercase singular noun of the entity being created (e.g. "customer"),
    // interpolated into the helper text.
    companyType: string
    // Organization types selectable as the attribution target.
    allowedTypes: string[]
    disabled?: boolean
    invalidMessage?: string
  }>(),
  {
    disabled: false,
    invalidMessage: '',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { t } = useI18n()
const loginStore = useLoginStore()

// Never offer the caller's own org: leaving the field empty already assigns the
// entity to it.
const excludeOrganizationIds = computed(() =>
  loginStore.userInfo?.organization_id ? [loginStore.userInfo.organization_id] : [],
)
</script>

<template>
  <OrganizationCombobox
    :model-value="props.modelValue"
    :label="t('organizations.created_on_behalf_of')"
    :optional="true"
    :allowed-types="props.allowedTypes"
    :exclude-organization-ids="excludeOrganizationIds"
    :disabled="props.disabled"
    :invalid-message="props.invalidMessage"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <template #tooltip>
      <NeTooltip>
        <template #content>{{
          t('organizations.created_on_behalf_of_helper', { companyType: props.companyType })
        }}</template>
      </NeTooltip>
    </template>
  </OrganizationCombobox>
</template>
