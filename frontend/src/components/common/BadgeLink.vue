<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeBadgeV2, type NeBadgeV2Kind } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { type IconDefinition } from '@fortawesome/free-solid-svg-icons'
import { RouterLink, type RouteLocationRaw } from 'vue-router'

const {
  to,
  kind,
  icon = undefined,
} = defineProps<{
  to: RouteLocationRaw
  kind: NeBadgeV2Kind
  icon?: IconDefinition
}>()
</script>

<template>
  <RouterLink :to="to" class="group">
    <NeBadgeV2 :kind="kind" class="group-hover:underline">
      <FontAwesomeIcon v-if="icon" :icon="icon" class="size-4" />
      <slot />
    </NeBadgeV2>
  </RouterLink>
</template>
