<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Wraps a table whose query keeps the previous rows while new ones load
  (placeholderData): during a reload an indeterminate bar runs over the top
  border of the table. The bar is out of the layout, so showing it never moves
  the table; the first load shows the skeleton of the table instead. It fades
  in only when the reload lasts longer than --duration-loading-delay.
-->

<script setup lang="ts">
import { NeProgressBar } from '@nethesis/vue-components'

const { updating = false } = defineProps<{
  updating?: boolean
}>()
</script>

<template>
  <div class="relative">
    <div
      v-if="updating"
      class="animate-delayed-fade-in absolute inset-x-0 top-0 z-10 overflow-hidden rounded-t-lg"
    >
      <NeProgressBar
        :progress="0"
        indeterminate
        size="sm"
        color="custom"
        custom-color-classes="bg-linear-to-r from-indigo-600 to-sky-500 dark:from-indigo-400 dark:to-sky-400"
      />
      <span class="sr-only">{{ $t('common.updating') }}</span>
    </div>
    <slot />
  </div>
</template>
