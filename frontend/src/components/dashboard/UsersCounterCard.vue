<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faUserGroup } from '@fortawesome/free-solid-svg-icons'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import CounterCard from '../CounterCard.vue'
import { getUsersTotal, USERS_TOTAL_KEY } from '@/lib/users/users'

const loginStore = useLoginStore()

const { state: usersTotal } = useQuery({
  key: [USERS_TOTAL_KEY],
  enabled: () => !!loginStore.jwtToken,
  query: getUsersTotal,
})
</script>

<template>
  <CounterCard
    :title="$t('users.title')"
    :counter="usersTotal.data ?? 0"
    :icon="faUserGroup"
    :loading="usersTotal.status === 'pending'"
  />
</template>
