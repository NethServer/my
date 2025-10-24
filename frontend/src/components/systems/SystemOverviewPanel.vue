<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeCard, NeSkeleton } from '@nethesis/vue-components'
import { useSystemDetail } from '@/queries/systems/systemDetail'

const { state, asyncStatus } = useSystemDetail()
</script>

<template>
  <div class="grid grid-cols-1 gap-x-6 gap-y-6 sm:grid-cols-2 2xl:grid-cols-4">
    <NeCard>
      <NeSkeleton v-if="state.status === 'pending'" :lines="10" />
      <div v-else>
        {{ state.data }}
      </div>
    </NeCard>
  </div>
</template>
