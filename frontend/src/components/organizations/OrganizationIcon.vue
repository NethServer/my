<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->
<script lang="ts" setup>
import {
  getOrganizationIcon,
  getOrganizationIconColorClasses,
} from '@/lib/organizations/organizations'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'

export type OrganizationIconSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl'
export type OrganizationIconVariant = 'avatar' | 'plain'

const {
  orgType,
  size = 'md',
  variant = 'avatar',
  colored = true,
} = defineProps<{
  orgType: string
  // in the avatar variant it sizes both the circle and the icon, in the plain
  // variant the icon alone
  size?: OrganizationIconSize
  // avatar: the icon on a gray circle; plain: the bare icon, inline
  variant?: OrganizationIconVariant
  // plain variant only: false drops the level color, so the icon takes the
  // color of its parent (e.g. a badge already colored by level)
  colored?: boolean
}>()

const avatarSizeClasses: Record<OrganizationIconSize, string> = {
  xs: 'size-6',
  sm: 'size-8',
  md: 'size-10',
  lg: 'size-12',
  xl: 'size-14',
  '2xl': 'size-16',
  '3xl': 'size-20',
  '4xl': 'size-24',
}

const iconSizeClasses: Record<OrganizationIconSize, string> = {
  xs: 'size-4',
  sm: 'size-4',
  md: 'size-5',
  lg: 'size-6',
  xl: 'size-7',
  '2xl': 'size-8',
  '3xl': 'size-10',
  '4xl': 'size-12',
}
</script>
<template>
  <FontAwesomeIcon
    v-if="variant === 'plain'"
    :icon="getOrganizationIcon(orgType)"
    :class="[colored && getOrganizationIconColorClasses(orgType), iconSizeClasses[size]]"
    aria-hidden="true"
  />
  <div v-else>
    <div
      :class="[
        'flex items-center justify-center rounded-full bg-gray-200 dark:bg-gray-700',
        avatarSizeClasses[size],
      ]"
    >
      <FontAwesomeIcon
        :icon="getOrganizationIcon(orgType)"
        :class="[getOrganizationIconColorClasses(orgType), iconSizeClasses[size]]"
      />
    </div>
  </div>
</template>
