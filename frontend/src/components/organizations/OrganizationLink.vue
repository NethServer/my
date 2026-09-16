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

// The three detail routes are gated on the level's read:* permission, with the
// own-organization exemption, so a link the API would refuse degrades to the
// plain name rather than to /forbidden.
const detailRoute = computed(() => {
  if (!canReadOrganizationDetail(organization.type, organization.logto_id ?? '')) {
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
