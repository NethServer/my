<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeHeading } from '@nethesis/vue-components'
import SupportSessionsTable from '@/components/support/SupportSessionsTable.vue'
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('support.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('support.page_description') }}
      </div>
    </div>
    <SupportSessionsTable />
  </div>
</template>
