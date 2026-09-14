<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import { organizationDetailRoute } from '@/lib/organizations/organizationDetailRoute'
import { canReadOrganizationDetail } from '@/lib/permissions'

/**
 * The organization of a creator snapshot, rendered as a link to its detail page.
 *
 * The link keeps the muted colour of the creator line it sits in and only
 * asserts itself on hover, so a table full of creators does not read as a wall
 * of links. Falls back to plain text when the organization has no detail page
 * (Owner, deleted, not synced yet) and when the current user could not open it.
 *
 * The permission check matters more here than on the other organization links:
 * a creator is frequently an upper tier, so a customer reading "On behalf of
 * <its distributor>" would otherwise be offered a page the API refuses it.
 */
const { organization } = defineProps<{
  organization: {
    organization_id: string
    organization_name: string
    organization_type?: string
  }
}>()

const detailRoute = computed(() => {
  if (
    !canReadOrganizationDetail(organization.organization_type ?? '', organization.organization_id)
  ) {
    return null
  }

  return organizationDetailRoute(organization.organization_id, organization.organization_type)
})
</script>

<template>
  <router-link
    v-if="detailRoute"
    :to="detailRoute"
    class="hover:text-primary-neutral cursor-pointer hover:underline"
  >
    {{ organization.organization_name }}
  </router-link>
  <template v-else>{{ organization.organization_name }}</template>
</template>
