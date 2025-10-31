<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faCrown } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeAvatar } from '@nethesis/vue-components'
import { computed } from 'vue'

export type AvatarSize = 'xxs' |'xs' | 'sm' | 'md' | 'lg'

const {
  size = 'md',
  isOwner,
  name,
} = defineProps<{
  size?: AvatarSize
  isOwner: boolean
  name: string
}>()

const circleSizeStyle: Record<AvatarSize, string> = {
  xxs: 'size-5',
  xs: 'size-6',
  sm: 'size-8',
  md: 'size-10',
  lg: 'size-12',
}

const iconSizeStyle: Record<AvatarSize, string> = {
  xxs: 'size-3',
  xs: 'size-3',
  sm: 'size-4',
  md: 'size-5',
  lg: 'size-6',
}

const userInitial = computed(() => {
  return name ? name.charAt(0).toUpperCase() : ''
})
</script>

<template>
  <div>
    <!-- owner avatar -->
    <NeAvatar v-if="isOwner" :size="size" aria-hidden="true">
      <template #placeholder>
        <div
          :class="`flex items-center justify-center rounded-full bg-gray-700 text-white dark:bg-gray-200 dark:text-gray-950 ${circleSizeStyle[size]}`"
        >
          <FontAwesomeIcon :icon="faCrown" :class="iconSizeStyle[size]" />
        </div>
      </template>
    </NeAvatar>
    <!-- avatar with initials -->
    <NeAvatar v-else :size="size" :initials="userInitial" aria-hidden="true" />
  </div>
</template>
