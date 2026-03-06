<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { getProductLogo, getProductName } from '@/lib/systems/systems'

const props = defineProps<{
  system?: string
}>()
</script>

<template>
  <img
    v-if="props.system"
    :src="getProductLogo(props.system)"
    :alt="getProductName(props.system)"
    aria-hidden="true"
    class="size-8 rounded-md"
  />
</template>
