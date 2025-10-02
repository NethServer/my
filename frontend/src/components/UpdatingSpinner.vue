<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeSpinner } from '@nethesis/vue-components'
</script>

<template>
  <div class="flex items-center gap-2">
    <NeSpinner color="white" />
    <div class="text-gray-500 dark:text-gray-400">
      {{ $t('common.updating') }}
    </div>
  </div>
</template>
