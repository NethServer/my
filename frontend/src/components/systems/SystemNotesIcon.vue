<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Next to a system name, shows the system notes in a tooltip. Long notes are
  cropped: the system details show them in full.
-->

<script setup lang="ts">
import { faCommentDots } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { NeTooltip } from '@nethesis/vue-components'

const { notes } = defineProps<{
  notes: string
}>()
</script>

<template>
  <NeTooltip trigger-event="mouseenter focus" placement="top">
    <template #trigger>
      <!-- a plain icon button, as the favorite star -->
      <button
        type="button"
        :aria-label="$t('common.notes')"
        class="text-secondary focus-visible:ring-primary-500 dark:focus-visible:ring-primary-300 flex rounded-sm p-1 transition-colors duration-(--duration-small) hover:text-indigo-900 focus-visible:ring-2 focus-visible:outline-hidden dark:hover:text-indigo-300"
      >
        <FontAwesomeIcon :icon="faCommentDots" class="size-4" aria-hidden="true" />
      </button>
    </template>
    <template #content>
      <p class="line-clamp-10 wrap-anywhere whitespace-pre-line">{{ notes }}</p>
    </template>
  </NeTooltip>
</template>
