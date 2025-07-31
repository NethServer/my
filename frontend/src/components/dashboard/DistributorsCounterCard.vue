<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faGlobe } from '@fortawesome/free-solid-svg-icons'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import { DISTRIBUTORS_TOTAL_KEY, getDistributorsTotal } from '@/lib/distributors'
import CounterCard from '../CounterCard.vue'

const loginStore = useLoginStore()

const { state: distributorsTotal } = useQuery({
  key: [DISTRIBUTORS_TOTAL_KEY],
  enabled: () => !!loginStore.jwtToken,
  query: getDistributorsTotal,
})
</script>

<template>
  <CounterCard
    :title="$t('distributors.title')"
    :counter="distributorsTotal.data ?? 0"
    :icon="faGlobe"
    :loading="distributorsTotal.status === 'pending'"
  />
  <!-- //// -->
  <!-- <NeCard>
    <NeSkeleton v-if="distributors.status === 'pending'" :lines="2" class="w-full" />
    <div v-else class="flex justify-between">
      <div class="flex items-center gap-3">
        <FontAwesomeIcon :icon="faGlobe" class="size-8 text-gray-600 dark:text-gray-300" />
        <NeHeading tag="h6" class="text-gray-600 dark:text-gray-300">
          {{ $t('distributors.title') }}
        </NeHeading>
      </div>
      <span class="text-3xl font-medium text-gray-900 dark:text-gray-50">
        {{ distributors.data?.length }}
      </span>
    </div>
  </NeCard> -->
</template>
