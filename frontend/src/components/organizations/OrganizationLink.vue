<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import { organizationDetailRoute } from '@/lib/organizations/organizationDetailRoute'
import { canReadOrganizationDetail } from '@/lib/permissions'

const { organization } = defineProps<{
  organization: {
    logto_id?: string
    name: string
    type: string
  }
}>()

// The three detail routes are gated on the level's read:* permission, so a
// company whose detail page the user cannot reach degrades to the plain name.
// This includes the user's own company: its level is the one level its org role
// carries no read:* for, so the page has no menu entry to return to.
const detailRoute = computed(() => {
  if (!canReadOrganizationDetail(organization.type)) {
    return null
  }

  return organizationDetailRoute(organization.logto_id, organization.type)
})
</script>

<template>
  <template v-if="detailRoute">
    <router-link :to="detailRoute!" class="cursor-pointer font-medium hover:underline">
      {{ organization.name || '-' }}
    </router-link>
  </template>
  <span v-else class="font-medium">
    {{ organization.name || '-' }}
  </span>
</template>
