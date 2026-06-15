<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faServer } from '@fortawesome/free-solid-svg-icons'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import CounterCard from '../CounterCard.vue'
import { NeBadgeV2 } from '@nethesis/vue-components'
import { getSystemsTotal, SYSTEMS_TOTAL_KEY } from '@/lib/systems/systems'
import { computed } from 'vue'

const loginStore = useLoginStore()

const { state: systemsTotal } = useQuery({
  key: [SYSTEMS_TOTAL_KEY],
  enabled: () => !!loginStore.jwtToken,
  query: getSystemsTotal,
})

const activeCount = computed(() => systemsTotal.value.data?.active ?? 0)
const inactiveCount = computed(() => systemsTotal.value.data?.inactive ?? 0)
const pendingCount = computed(() => systemsTotal.value.data?.unknown ?? 0)
</script>

<template>
  <CounterCard
    :title="$t('systems.total_systems')"
    :counter="systemsTotal.data?.total ?? 0"
    :icon="faServer"
    :loading="systemsTotal.status === 'pending'"
    title-route-name="systems"
  >
    <div class="mt-5 flex flex-wrap justify-center gap-2">
      <NeBadgeV2 v-if="activeCount > 0" kind="green">
        {{ $t('systems.count_active', { count: activeCount }) }}
      </NeBadgeV2>
      <NeBadgeV2 v-if="inactiveCount > 0" kind="rose">
        {{ $t('systems.count_inactive', { count: inactiveCount }) }}
      </NeBadgeV2>
      <NeBadgeV2 v-if="pendingCount > 0" kind="gray">
        {{ $t('systems.count_pending', { count: pendingCount }) }}
      </NeBadgeV2>
    </div>
  </CounterCard>
</template>
