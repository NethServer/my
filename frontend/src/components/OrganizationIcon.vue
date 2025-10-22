<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->
<script lang="ts" setup>
import { faBuilding, faCity, faGlobe, faQuestion } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { computed } from 'vue'

type Size = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl'

const {
  orgType,
  size = 'md',
  squared = false,
} = defineProps<{
  orgType: string
  size?: Size
  squared?: boolean
}>()

const avatarSizeClasses: Record<Size, string> = {
  xs: 'size-6',
  sm: 'size-8',
  md: 'size-10',
  lg: 'size-12',
  xl: 'size-14',
  '2xl': 'size-16',
  '3xl': 'size-20',
  '4xl': 'size-24',
}

const placeholderColorClasses = 'bg-gray-700 text-white dark:bg-gray-200 dark:text-gray-950'

const placeholderIconSizeClasses: Record<Size, string> = {
  xs: 'size-3',
  sm: 'size-4',
  md: 'size-5',
  lg: 'size-6',
  xl: 'size-7',
  '2xl': 'size-8',
  '3xl': 'size-10',
  '4xl': 'size-12',
}

const placeholderContainerClasses = computed(
  () =>
    `flex items-center justify-center ${placeholderColorClasses} ${squared ? 'rounded-sm' : 'rounded-full'} ${avatarSizeClasses[size]}`,
)

function getOrganizationIcon() {
  switch (orgType) {
    case 'distributor':
      return faGlobe
    case 'reseller':
      return faCity
    case 'customer':
      return faBuilding
    default:
      return faQuestion
  }
}
</script>
<template>
  <div>
    <div :class="placeholderContainerClasses">
      <FontAwesomeIcon
        :icon="getOrganizationIcon()"
        :class="[placeholderColorClasses, placeholderIconSizeClasses[size]]"
      />
    </div>
  </div>
</template>
