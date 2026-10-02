<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeTooltip } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'
import OrganizationIcon from '@/components/organizations/OrganizationIcon.vue'
import OrganizationLink from '@/components/organizations/OrganizationLink.vue'
import {
  type OrganizationIconSize,
  type OrganizationIconVariant,
} from '@/components/organizations/OrganizationIcon.vue'

const {
  organization,
  iconSize = 'sm',
  iconVariant = 'avatar',
} = defineProps<{
  organization: {
    logto_id?: string
    name: string
    type: string
  }
  iconSize?: OrganizationIconSize
  iconVariant?: OrganizationIconVariant
}>()

const { t } = useI18n()
</script>

<template>
  <div class="inline wrap-anywhere">
    <NeTooltip
      v-if="organization.type"
      placement="top"
      trigger-event="mouseenter focus"
      class="mr-2 inline-block align-middle"
    >
      <template #trigger>
        <OrganizationIcon :org-type="organization.type" :size="iconSize" :variant="iconVariant" />
      </template>
      <template #content>
        {{ t(`organizations.${organization.type.toLowerCase()}`) }}
      </template>
    </NeTooltip>
    <OrganizationLink :organization="organization" />
  </div>
</template>
