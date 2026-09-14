<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import { organizationDetailRoute } from '@/lib/organizations/organizationDetailRoute'

const { organization } = defineProps<{
  organization: {
    logto_id?: string
    name: string
    type: string
  }
}>()

const detailRoute = computed(() =>
  organizationDetailRoute(organization.logto_id, organization.type),
)
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
