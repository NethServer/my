<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeHeading, NeSkeleton, NeTooltip } from '@nethesis/vue-components'
import { type IconDefinition } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { computed, useSlots } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, type RouteRecordNameGeneric } from 'vue-router'
import { abbreviateNumber } from '@/lib/common'

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
  abbreviateCounter = true,
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
  abbreviateCounter?: boolean
}>()

const ABBREVIATION_THRESHOLD = 10_000

const { locale } = useI18n()

const abbreviatedCounter = computed(() => {
  if (abbreviateCounter) {
    return abbreviateNumber(counter, locale.value, ABBREVIATION_THRESHOLD)
  }
  return counter
})

const formattedCounter = computed(() => {
  return new Intl.NumberFormat(locale.value).format(counter)
})

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
    <template v-if="!centeredCounter" #topRight>
      <div
        :class="['text-4xl font-medium', colorClasses ?? 'text-indigo-700 dark:text-indigo-500']"
      >
        <NeTooltip v-if="counter >= ABBREVIATION_THRESHOLD" trigger-event="mouseenter focus">
          <template #trigger>
            <span> {{ isEstimated ? '~' : '' }}{{ abbreviatedCounter }} </span>
          </template>
          <template #content>
            {{ formattedCounter }}
          </template>
        </NeTooltip>
        <span v-else> {{ isEstimated ? '~' : '' }}{{ abbreviatedCounter }} </span>
      </div>
    </template>
    <NeSkeleton v-if="loading" :lines="skeletonLines" class="w-full" />
    <template v-else>
      <div v-if="centeredCounter" class="flex flex-col gap-4">
        <div
          :class="[
            'self-center text-4xl font-medium',
            colorClasses ?? 'text-indigo-700 dark:text-indigo-500',
          ]"
        >
          <NeTooltip v-if="counter >= ABBREVIATION_THRESHOLD" trigger-event="mouseenter focus">
            <template #trigger>
              <span> {{ isEstimated ? '~' : '' }}{{ abbreviatedCounter }} </span>
            </template>
            <template #content>
              {{ formattedCounter }}
            </template>
          </NeTooltip>
          <span v-else> {{ isEstimated ? '~' : '' }}{{ abbreviatedCounter }} </span>
        </div>
      </div>
      <div v-if="hasDefaultSlot" class="mt-5">
        <slot></slot>
      </div>
    </template>
  </NeCard>
</template>
