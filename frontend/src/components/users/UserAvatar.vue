<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { API_URL } from '@/lib/config'
import { faCrown } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeAvatar, type AvatarSize } from '@nethesis/vue-components'
import { computed } from 'vue'

const {
  size = 'md',
  isOwner,
  name,
  logtoId,
  cacheKey,
} = defineProps<{
  size?: AvatarSize
  isOwner: boolean
  name: string
  logtoId: string
  cacheKey?: number | string
}>()

const circleSizeStyle: Record<AvatarSize, string> = {
  xs: 'size-6',
  sm: 'size-8',
  md: 'size-10',
  lg: 'size-12',
  xl: 'size-14',
  '2xl': 'size-16',
  '3xl': 'size-20',
  '4xl': 'size-24',
}

const iconSizeStyle: Record<AvatarSize, string> = {
  xs: 'size-3',
  sm: 'size-4',
  md: 'size-5',
  lg: 'size-6',
  xl: 'size-7',
  '2xl': 'size-8',
  '3xl': 'size-10',
  '4xl': 'size-12',
}

const userInitial = computed(() => {
  return name ? name.charAt(0).toUpperCase() : ''
})

const avatarUrl = computed(() => {
  const url = `${API_URL}/public/users/${logtoId}/avatar`
  return cacheKey ? `${url}?v=${encodeURIComponent(String(cacheKey))}` : url
})
</script>

<template>
  <!-- owner avatar -->
  <NeAvatar
    v-if="isOwner"
    :key="`owner-${cacheKey}`"
    :size="size"
    aria-hidden="true"
    :img="avatarUrl"
  >
    <template #placeholder>
      <div
        :class="`flex items-center justify-center rounded-full bg-gray-700 text-white dark:bg-gray-200 dark:text-gray-950 ${circleSizeStyle[size]}`"
      >
        <FontAwesomeIcon :icon="faCrown" :class="iconSizeStyle[size]" />
      </div>
    </template>
  </NeAvatar>
  <!-- avatar with initials -->
  <NeAvatar
    v-else
    :key="`user-${cacheKey}`"
    :size="size"
    :initials="userInitial"
    aria-hidden="true"
    :img="avatarUrl"
  />
</template>
