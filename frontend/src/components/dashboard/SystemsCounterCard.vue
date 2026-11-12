<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faServer } from '@fortawesome/free-solid-svg-icons'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import CounterCard from '../common/CounterCard.vue'
import { getSystemsTotal, SYSTEMS_TOTAL_KEY } from '@/lib/systems/systems'
import { computed } from 'vue'

const loginStore = useLoginStore()

const { state: systemsTotal } = useQuery({
  key: [SYSTEMS_TOTAL_KEY],
  enabled: () => !!loginStore.jwtToken,
  query: getSystemsTotal,
})

const totalCount = computed(() => systemsTotal.value.data?.total ?? 0)
</script>

<template>
  <CounterCard
    :title="$t('systems.total_systems')"
    :counter="totalCount"
    :icon="faServer"
    :loading="systemsTotal.status === 'pending'"
    :to="{ name: 'systems' }"
  />
</template>
