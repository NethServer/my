<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faBuilding, faCity, faGlobe, faPalette } from '@fortawesome/free-solid-svg-icons'
import { NeInlineNotification } from '@nethesis/vue-components'
import { computed } from 'vue'
import CounterCard from '@/components/common/CounterCard.vue'
import { useRebrandingSummary } from '@/queries/rebranding/rebrandingSummary'

const { state } = useRebrandingSummary()

const loading = computed(() => state.value.status === 'pending')
const summary = computed(() => state.value.data)
</script>

<template>
  <div>
    <NeInlineNotification
      v-if="state.status === 'error'"
      kind="error"
      :title="$t('rebranding.cannot_retrieve_rebranding_summary')"
      :description="state.error.message"
      class="mb-6"
    />
    <div class="grid grid-cols-1 gap-x-6 gap-y-6 sm:grid-cols-2 2xl:grid-cols-4">
      <CounterCard
        :title="$t('rebranding.companies_with_rebranding')"
        :counter="summary?.total ?? 0"
        :icon="faPalette"
        :loading="loading"
      />
      <CounterCard
        :title="$t('distributors.title')"
        :counter="summary?.distributors ?? 0"
        :icon="faGlobe"
        :loading="loading"
      />
      <CounterCard
        :title="$t('resellers.title')"
        :counter="summary?.resellers ?? 0"
        :icon="faCity"
        :loading="loading"
      />
      <CounterCard
        :title="$t('customers.title')"
        :counter="summary?.customers ?? 0"
        :icon="faBuilding"
        :loading="loading"
      />
    </div>
  </div>
</template>
