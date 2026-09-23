<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { computed } from 'vue'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faServer } from '@fortawesome/free-solid-svg-icons'
import { getProductLogo, getProductName } from '@/lib/systems/systems'

export type SystemLogoSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl'

const { system, size = 'sm' } = defineProps<{
  system?: string
  size?: SystemLogoSize
}>()

const sizeClasses: Record<SystemLogoSize, string> = {
  xs: 'size-6',
  sm: 'size-8',
  md: 'size-10',
  lg: 'size-12',
  xl: 'size-14',
  '2xl': 'size-16',
  '3xl': 'size-20',
  '4xl': 'size-24',
}

// Same steps as OrganizationIcon, for the placeholder shown until the system
// reports its type
const placeholderIconSizeClasses: Record<SystemLogoSize, string> = {
  xs: 'size-4',
  sm: 'size-4',
  md: 'size-5',
  lg: 'size-6',
  xl: 'size-7',
  '2xl': 'size-8',
  '3xl': 'size-10',
  '4xl': 'size-12',
}

const logo = computed(() => (system ? getProductLogo(system) : undefined))
</script>

<template>
  <img
    v-if="logo"
    :src="logo"
    :alt="getProductName(system!)"
    aria-hidden="true"
    :class="`${sizeClasses[size]} rounded-md`"
  />
  <!-- no type yet (or an unknown one) -->
  <div
    v-else
    :class="`${sizeClasses[size]} flex shrink-0 items-center justify-center rounded-md bg-gray-700 text-white dark:bg-gray-200 dark:text-gray-950`"
    aria-hidden="true"
  >
    <FontAwesomeIcon :icon="faServer" :class="placeholderIconSizeClasses[size]" />
  </div>
</template>
