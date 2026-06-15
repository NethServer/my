<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
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
</script>

<template>
  <img
    v-if="system"
    :src="getProductLogo(system)"
    :alt="getProductName(system)"
    aria-hidden="true"
    :class="`${sizeClasses[size]} rounded-md`"
  />
</template>
