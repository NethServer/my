<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeHeading, NeSkeleton } from '@nethesis/vue-components'
import { type IconDefinition } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { computed, useSlots } from 'vue'

const {
  title,
  counter,
  icon = undefined,
  loading = false,
  skeletonLines = 2,
} = defineProps<{
  title: string
  counter: number
  icon?: IconDefinition
  loading?: boolean
  skeletonLines?: number
}>()

const slots = useSlots()

const hasDefaultSlot = computed(() => !!slots.default)
</script>

<template>
  <NeCard>
    <NeSkeleton v-if="loading" :lines="skeletonLines" class="w-full" />
    <template v-else>
      <div class="flex justify-between">
        <div class="flex items-center gap-3">
          <FontAwesomeIcon
            v-if="icon"
            :icon="icon"
            class="size-5 text-gray-600 dark:text-gray-300"
          />
          <NeHeading tag="h6" class="text-gray-600 dark:text-gray-300">
            {{ title }}
          </NeHeading>
        </div>
        <span class="text-3xl font-medium text-gray-900 dark:text-gray-50">
          {{ counter }}
        </span>
      </div>
      <div v-if="hasDefaultSlot" class="mt-5">
        <slot></slot>
      </div>
    </template>
  </NeCard>
</template>
