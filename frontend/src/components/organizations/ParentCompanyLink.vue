<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import OrganizationIconAndLink from '@/components/organizations/OrganizationIconAndLink.vue'

/**
 * The company an organization belongs to, rendered as a link to its detail
 * page.
 *
 * The parent company is the organization the entity was attributed to at
 * creation (`custom_data.createdBy`), which the creator snapshot already
 * carries: the backend stamps the attributed organization on it, so this is
 * the same company the parent company list filter matches on.
 *
 * OrganizationIconAndLink drops the link (and the level icon) on its own when
 * the organization has no detail page or the user may not read it, so the Owner
 * organization and an out-of-scope parent degrade to the plain name.
 */
const { creator = undefined } = defineProps<{
  creator?: {
    organization_id: string
    organization_name: string
    organization_type?: string
  }
}>()

const organization = computed(() => {
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
  <OrganizationIconAndLink v-if="organization" :organization="organization" icon-size="xs" />
  <template v-else>-</template>
</template>
