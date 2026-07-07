<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { RouterLink } from 'vue-router'
import SystemLogo, { type SystemLogoSize } from '@/components/systems/SystemLogo.vue'
import { NeTooltip } from '@nethesis/vue-components'
import { getProductName } from '@/lib/systems/systems'

const { size = 'sm' } = defineProps<{
  systemId?: string
  systemName?: string
  systemType?: string
  size?: SystemLogoSize
}>()
</script>

<template>
  <div class="flex items-center gap-2">
    <span v-if="systemType" class="shrink-0">
      <NeTooltip trigger-event="mouseenter focus">
        <template #trigger>
          <SystemLogo :system="systemType" :size="size" />
        </template>
        <template #content>
          {{ getProductName(systemType) }}
        </template>
      </NeTooltip>
    </span>
    <RouterLink
      v-if="systemId"
      :to="{ name: 'system_detail', params: { systemId } }"
      class="cursor-pointer font-medium hover:underline"
    >
      {{ systemName || '-' }}
    </RouterLink>
    <span v-else class="text-tertiary-neutral">
      {{ systemName || '-' }}
    </span>
  </div>
</template>
