<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faServer } from '@fortawesome/free-solid-svg-icons'
import { NeLink } from '@nethesis/vue-components'
import { useQuery } from '@pinia/colada'
import { computed } from 'vue'
import { useLoginStore } from '@/stores/login'
import CounterCard from '../common/CounterCard.vue'
import { getSystemsTotal, SYSTEMS_TOTAL_KEY } from '@/lib/systems/systems'

const legacySystemsUrl = 'https://legacy.my.nethesis.it'

const loginStore = useLoginStore()

const { state: systemsTotal } = useQuery({
  key: [SYSTEMS_TOTAL_KEY],
  enabled: () => !!loginStore.jwtToken,
  query: getSystemsTotal,
})

const legacyCount = computed(() => systemsTotal.value.data?.legacy ?? 0)
</script>

<template>
  <!--
    Systems still on the old my are a migration leftover: the card is only worth
    a grid slot while the partner has some, so it stays hidden at zero (and
    while the count is still loading, to avoid a card that appears then leaves).
  -->
  <CounterCard
    v-if="legacyCount > 0"
    :title="$t('systems.total_legacy_systems')"
    :counter="legacyCount"
    :icon="faServer"
  >
    <template #title-tooltip>
      <i18n-t keypath="systems.total_legacy_systems_tooltip" tag="span" scope="global">
        <template #url>
          <NeLink :href="legacySystemsUrl" target="_blank" rel="noopener noreferrer">
            {{ legacySystemsUrl }}
          </NeLink>
        </template>
      </i18n-t>
    </template>
  </CounterCard>
</template>
