<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  An action inside a tooltip, such as "Extend session". It looks like an
  NeLink with inverted-theme, readable on the dark tooltip, but it is a
  button: an NeLink without href renders an <a> that the keyboard skips, and
  NeButton has no variant for the tooltip background.
-->

<script setup lang="ts">
defineEmits<{ click: [] }>()
</script>

<template>
  <button
    type="button"
    class="text-primary-500 hover:text-primary-400 dark:text-primary-700 dark:hover:text-primary-800 focus:ring-primary-500 cursor-pointer rounded-sm font-medium hover:underline focus:ring-2 focus:ring-offset-2 focus:outline-hidden"
    @click="$emit('click')"
  >
    <slot></slot>
  </button>
</template>
