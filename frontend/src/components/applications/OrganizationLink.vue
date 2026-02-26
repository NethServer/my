<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'

const { organization } = defineProps<{
  organization: {
    logto_id?: string
    name: string
    type: string
  }
}>()

const organizationDetailRoute = computed(() => {
  if (!organization.logto_id) return null

  const lowerType = organization.type.toLowerCase()

  switch (lowerType) {
    case 'distributor':
      return {
        name: 'distributor_detail',
        params: { companyId: organization.logto_id },
      }
    case 'reseller':
      return {
        name: 'reseller_detail',
        params: { companyId: organization.logto_id },
      }
    case 'customer':
      return {
        name: 'customer_detail',
        params: { companyId: organization.logto_id },
      }
    default:
      return null
  }
})
</script>

<template>
  <template v-if="organizationDetailRoute">
    <router-link :to="organizationDetailRoute!" class="cursor-pointer font-medium hover:underline">
      {{ organization.name || '-' }}
    </router-link>
  </template>
  <span v-else class="font-medium">
    {{ organization.name || '-' }}
  </span>
</template>
