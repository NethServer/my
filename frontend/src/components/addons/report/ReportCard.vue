<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  A panel of the add-on report. NeCard's own title renders dark and prominent;
  the report's headings are the muted, uppercase kind CounterCard uses, so its
  wide panels and the four counters above them read as one board rather than
  two designs.
-->

<script setup lang="ts">
import { NeCard, NeHeading, NeSkeleton } from '@nethesis/vue-components'

const { title, loading = false } = defineProps<{
  title: string
  loading?: boolean
}>()
</script>

<template>
  <NeCard>
    <template #title>
      <div class="text-tertiary-neutral dark:text-tertiary-neutral">
        <NeHeading tag="h6" class="text-inherit">{{ title.toUpperCase() }}</NeHeading>
      </div>
    </template>
    <NeSkeleton v-if="loading" :lines="8" class="mt-4 w-full" />
    <div v-else class="mt-4">
      <slot />
    </div>
  </NeCard>
</template>
