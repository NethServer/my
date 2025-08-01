<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faCity } from '@fortawesome/free-solid-svg-icons'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import { getResellersTotal, RESELLERS_TOTAL_KEY } from '@/lib/resellers'
import CounterCard from '../CounterCard.vue'

const loginStore = useLoginStore()

const { state: resellersTotal } = useQuery({
  key: [RESELLERS_TOTAL_KEY],
  enabled: () => !!loginStore.jwtToken,
  query: getResellersTotal,
})
</script>

<template>
  <CounterCard
    :title="$t('resellers.title')"
    :counter="resellersTotal.data ?? 0"
    :icon="faCity"
    :loading="resellersTotal.status === 'pending'"
  />
</template>
