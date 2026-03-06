<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { getApplicationLogo } from '@/lib/applications/applications'

const props = defineProps<{
  app?: string
}>()
</script>

<template>
  <img
    v-if="props.app"
    :src="getApplicationLogo(props.app)"
    :alt="props.app"
    aria-hidden="true"
    class="size-8 rounded-md"
  />
</template>
