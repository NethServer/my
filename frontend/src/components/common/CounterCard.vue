<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeHeading, NeSkeleton, NeTooltip } from '@nethesis/vue-components'
import { type IconDefinition } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { computed, useAttrs, useSlots } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import { abbreviateNumber } from '@/lib/common'

// The counter number carries the interactivity, not the card: `to` renders it
// as a RouterLink, and a `@counter-click` listener renders it as a button.
// inheritAttrs is off so that listener lands on the number, while layout attrs
// (class, etc.) are forwarded to NeCard via containerAttrs.
defineOptions({ inheritAttrs: false })

const {
  title,
  counter,
  icon = undefined,
  loading = false,
  skeletonLines = 2,
  uppercaseTitle = true,
  centeredCounter = true,
  colorClasses = undefined,
  to = undefined,
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
  to?: RouteLocationRaw
  abbreviateCounter?: boolean
}>()

const attrs = useAttrs()

const isClickable = computed(() => typeof attrs.onCounterClick === 'function')
const isInteractive = computed(() => to != null || isClickable.value)

const containerAttrs = computed(() => {
  const rest: Record<string, unknown> = { ...attrs }
  delete rest.onCounterClick
  return rest
})

const counterTag = computed(() => (to != null ? RouterLink : isClickable.value ? 'button' : 'div'))

const counterAttrs = computed(() => {
  if (to != null) {
    return { to, 'aria-label': title }
  }
  if (isClickable.value) {
    return { type: 'button', 'aria-label': title, onClick: attrs.onCounterClick }
  }
  return {}
})

// The title mirrors the counter's interactivity: a RouterLink when `to` is set,
// a button when `@counter-click` is listened to, a plain heading otherwise.
const titleTag = computed(() => (to != null ? RouterLink : isClickable.value ? 'button' : 'div'))

const titleAttrs = computed(() => {
  if (to != null) {
    return { to }
  }
  if (isClickable.value) {
    return { type: 'button', onClick: attrs.onCounterClick }
  }
  return {}
})

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
  <NeCard v-bind="containerAttrs">
    <template #title>
      <div class="flex items-center gap-3">
        <FontAwesomeIcon
          v-if="icon"
          :icon="icon"
          class="text-tertiary-neutral dark:text-tertiary-neutral size-5"
        />
        <component
          :is="titleTag"
          v-bind="titleAttrs"
          :class="[
            'text-tertiary-neutral dark:text-tertiary-neutral',
            isInteractive && 'cursor-pointer hover:underline',
          ]"
        >
          <NeHeading tag="h6" class="text-inherit">
            {{ uppercaseTitle ? title.toUpperCase() : title }}
          </NeHeading>
        </component>
      </div>
    </template>
    <template v-if="!centeredCounter" #topRight>
      <component
        :is="counterTag"
        v-bind="counterAttrs"
        :class="[
          'text-4xl font-medium',
          isInteractive && 'cursor-pointer',
          colorClasses ?? 'text-indigo-700 dark:text-indigo-500',
        ]"
      >
        <NeTooltip v-if="counter >= ABBREVIATION_THRESHOLD" trigger-event="mouseenter focus">
          <template #trigger>
            <span> {{ abbreviatedCounter }} </span>
          </template>
          <template #content>
            {{ formattedCounter }}
          </template>
        </NeTooltip>
        <span v-else> {{ abbreviatedCounter }} </span>
      </component>
    </template>
    <NeSkeleton v-if="loading" :lines="skeletonLines" class="w-full" />
    <template v-else>
      <div v-if="centeredCounter" class="flex flex-col gap-4">
        <component
          :is="counterTag"
          v-bind="counterAttrs"
          :class="[
            'self-center text-4xl font-medium',
            isInteractive && 'cursor-pointer',
            colorClasses ?? 'text-indigo-700 dark:text-indigo-500',
          ]"
        >
          <NeTooltip v-if="counter >= ABBREVIATION_THRESHOLD" trigger-event="mouseenter focus">
            <template #trigger>
              <span> {{ abbreviatedCounter }} </span>
            </template>
            <template #content>
              {{ formattedCounter }}
            </template>
          </NeTooltip>
          <span v-else> {{ abbreviatedCounter }} </span>
        </component>
      </div>
      <div v-if="hasDefaultSlot" class="mt-5">
        <slot></slot>
      </div>
    </template>
  </NeCard>
</template>
