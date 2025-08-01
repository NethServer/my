<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faBuilding } from '@fortawesome/free-solid-svg-icons'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import { CUSTOMERS_TOTAL_KEY, getCustomersTotal } from '@/lib/customers'
import CounterCard from '../CounterCard.vue'

const loginStore = useLoginStore()

const { state: customersTotal } = useQuery({
  key: [CUSTOMERS_TOTAL_KEY],
  enabled: () => !!loginStore.jwtToken,
  query: getCustomersTotal,
})
</script>

<template>
  <CounterCard
    :title="$t('customers.title')"
    :counter="customersTotal.data ?? 0"
    :icon="faBuilding"
    :loading="customersTotal.status === 'pending'"
  />
</template>
