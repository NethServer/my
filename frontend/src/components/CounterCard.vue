<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeHeading, NeSkeleton } from '@nethesis/vue-components'
import { type IconDefinition } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { computed, useSlots } from 'vue'
import { RouterLink, type RouteRecordNameGeneric } from 'vue-router'

const {
  title,
  counter,
  icon = undefined,
  loading = false,
  skeletonLines = 2,
  uppercaseTitle = true,
  centeredCounter = true,
  colorClasses = undefined,
  isEstimated = false,
  titleRouteName = undefined,
} = defineProps<{
  title: string
  counter: number
  icon?: IconDefinition
  loading?: boolean
  skeletonLines?: number
  uppercaseTitle?: boolean
  centeredCounter?: boolean
  colorClasses?: string
  isEstimated?: boolean
  titleRouteName?: RouteRecordNameGeneric
}>()

const slots = useSlots()

const hasDefaultSlot = computed(() => !!slots.default)
</script>

<template>
  <NeCard>
    <template #title>
      <div class="flex items-center gap-3">
        <FontAwesomeIcon
          v-if="icon"
          :icon="icon"
          class="text-tertiary-neutral dark:text-tertiary-neutral size-5"
        />
        <RouterLink
          v-if="titleRouteName"
          :to="{ name: titleRouteName }"
          class="text-tertiary-neutral dark:text-tertiary-neutral cursor-pointer hover:underline"
        >
          <NeHeading tag="h6" class="text-inherit">
            {{ uppercaseTitle ? title.toUpperCase() : title }}
          </NeHeading>
        </RouterLink>
        <NeHeading v-else tag="h6" class="text-tertiary-neutral dark:text-tertiary-neutral">
          {{ uppercaseTitle ? title.toUpperCase() : title }}
        </NeHeading>
      </div>
    </template>
    <NeSkeleton v-if="loading" :lines="skeletonLines" class="w-full" />
    <template v-else>
      <div :class="['flex', centeredCounter ? 'flex-col gap-4' : 'justify-between']">
        <span
          :class="[
            'self-center text-4xl font-medium',
            colorClasses ?? 'text-indigo-700 dark:text-indigo-500',
            { 'self-center': centeredCounter },
          ]"
        >
          {{ isEstimated ? '~' : '' }}{{ counter }}
        </span>
      </div>
      <div v-if="hasDefaultSlot" class="mt-5">
        <slot></slot>
      </div>
    </template>
  </NeCard>
</template>
