<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import OrganizationIconAndLink from '@/components/organizations/OrganizationIconAndLink.vue'
import { type OrganizationIconSize } from '@/components/organizations/OrganizationIcon.vue'
import type { ParentOrganization } from '@/lib/organizations/organizations'

/**
 * The company an organization belongs to, rendered as a link to its detail
 * page.
 *
 * The parent company is the organization the entity was attributed to at
 * creation (`custom_data.createdBy`). It comes either as `parent`, resolved by
 * the systems endpoints for the company a row is assigned to, or
 * from the creator snapshot of a distributor, reseller or customer: the backend
 * stamps the attributed organization on it, so this is the same company the
 * parent company list filter matches on.
 *
 * OrganizationIconAndLink drops the link (and the level icon) on its own when
 * the organization has no detail page or the user may not read it, so the Owner
 * organization and an out-of-scope parent degrade to the plain name.
 */
const {
  creator = undefined,
  parent = undefined,
  iconSize = 'sm',
} = defineProps<{
  creator?: {
    organization_id: string
    organization_name: string
    organization_type?: string
  }
  parent?: ParentOrganization
  iconSize?: OrganizationIconSize
}>()

const { t } = useI18n()

const organization = computed(() => {
  if (parent) {
    return {
      logto_id: parent.logto_id,
      // the database may not record the Owner organization's name
      name: parent.name || t('organizations.owner'),
      type: parent.type,
    }
  }

  if (!creator?.organization_name) {
    return null
  }

  return {
    logto_id: creator.organization_id,
    name: creator.organization_name,
    type: creator.organization_type ?? '',
  }
})
</script>

<template>
  <OrganizationIconAndLink v-if="organization" :organization="organization" :icon-size="iconSize" />
  <template v-else>-</template>
</template>
