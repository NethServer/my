<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  The shell every non-counter dashboard widget sits in: a NeCard with a title, a
  skeleton while the data loads and an inline error when it fails.

  It carries no height of its own. Filling the grid cell is done from
  src/assets/gridstack.css, which has to reach inside NeCard to do it — see the
  comment there.
-->

<script setup lang="ts">
import { NeCard, NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'

const {
  title,
  loading = false,
  error = undefined,
  skeletonLines = 6,
} = defineProps<{
  title: string
  loading?: boolean
  error?: string
  skeletonLines?: number
}>()
</script>

<template>
  <NeCard>
    <template #title>
      <div class="text-tertiary-neutral dark:text-tertiary-neutral">
        <NeHeading tag="h6" class="text-inherit">{{ title.toUpperCase() }}</NeHeading>
      </div>
    </template>
    <NeSkeleton v-if="loading" :lines="skeletonLines" class="mt-4 w-full" />
    <NeInlineNotification
      v-else-if="error"
      kind="error"
      :title="$t('dashboard.widget_failed')"
      :description="error"
      class="mt-4"
    />
    <div v-else class="mt-4">
      <slot />
    </div>
  </NeCard>
</template>
