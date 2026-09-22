<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { useQuery } from '@pinia/colada'
import { computed } from 'vue'
import { useLoginStore } from '@/stores/login'
import LegacySystemsCard from '../systems/LegacySystemsCard.vue'
import { getSystemsTotal, SYSTEMS_TOTAL_KEY } from '@/lib/systems/systems'

const loginStore = useLoginStore()

const { state: systemsTotal } = useQuery({
  key: [SYSTEMS_TOTAL_KEY],
  enabled: () => !!loginStore.jwtToken,
  query: getSystemsTotal,
})

const legacyCount = computed(() => systemsTotal.value.data?.legacy ?? 0)
</script>

<template>
  <LegacySystemsCard :counter="legacyCount" :loading="systemsTotal.status === 'pending'" />
</template>
