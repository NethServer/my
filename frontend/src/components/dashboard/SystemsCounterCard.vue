<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faServer } from '@fortawesome/free-solid-svg-icons'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import CounterCard from '../CounterCard.vue'
import { getSystemsTotal, SYSTEMS_TOTAL_KEY } from '@/lib/systems/systems'

const loginStore = useLoginStore()

const { state: systemsTotal } = useQuery({
  key: [SYSTEMS_TOTAL_KEY],
  enabled: () => !!loginStore.jwtToken,
  query: getSystemsTotal,
})
</script>

<template>
  <CounterCard
    :title="$t('systems.title')"
    :counter="systemsTotal.data ?? 0"
    :icon="faServer"
    :loading="systemsTotal.status === 'pending'"
  />
</template>
