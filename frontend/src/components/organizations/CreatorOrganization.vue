<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import CreatorOrganizationLink from '@/components/organizations/CreatorOrganizationLink.vue'

/**
 * The organization line under a creator: either the organization the creator
 * belongs to, or "On behalf of <organization>" when the entity was attributed
 * to another organization at creation. The name links to the organization
 * detail page in both cases.
 */
const { creator } = defineProps<{
  creator: {
    organization_id: string
    organization_name: string
    organization_type?: string
    on_behalf_of?: boolean
  }
}>()
</script>

<template>
  <i18n-t v-if="creator.on_behalf_of" keypath="systems.on_behalf_of" tag="span">
    <template #organization>
      <CreatorOrganizationLink :organization="creator" />
    </template>
  </i18n-t>
  <CreatorOrganizationLink v-else :organization="creator" />
</template>
