<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faGlobe } from '@fortawesome/free-solid-svg-icons'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import { DISTRIBUTORS_TOTAL_KEY, getDistributorsTotal } from '@/lib/organizations/distributors'
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
</template>
