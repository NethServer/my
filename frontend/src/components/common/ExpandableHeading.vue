<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeExpandable, NeHeading } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faChevronDown, faChevronUp } from '@fortawesome/free-solid-svg-icons'
import { ref } from 'vue'

/**
 * A section heading that shows and hides the content below it, closed at
 * first. NeExpandable puts its click handler on a div around the trigger: the
 * button inside the heading is what the keyboard reaches, and its click is the
 * one the div receives.
 */
const { label } = defineProps<{
  label: string
}>()

const isExpanded = ref(false)
</script>

<template>
  <NeExpandable @set-expanded="isExpanded = $event">
    <template #trigger>
      <NeHeading tag="h6">
        <button
          type="button"
          :aria-expanded="isExpanded"
          class="focus:ring-primary-500 dark:focus:ring-primary-300 flex items-center gap-2 rounded-sm text-left focus:ring-2 focus:ring-offset-2 focus:ring-offset-white focus:outline-hidden dark:focus:ring-offset-gray-900"
        >
          {{ label }}
          <FontAwesomeIcon
            :icon="isExpanded ? faChevronUp : faChevronDown"
            class="size-3 shrink-0"
            aria-hidden="true"
          />
        </button>
      </NeHeading>
    </template>
    <slot />
  </NeExpandable>
</template>
