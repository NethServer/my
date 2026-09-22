<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faServer } from '@fortawesome/free-solid-svg-icons'
import { NeLink } from '@nethesis/vue-components'
import CounterCard from '../common/CounterCard.vue'
import { LEGACY_SYSTEMS_URL } from '@/lib/systems/systems'

/**
 * The counter of the systems still on the old my, with the tooltip linking to
 * it. The counter itself comes from the caller, which knows whether it is the
 * signed-in partner's total or an organization's hierarchy count.
 */
const { counter, loading = false } = defineProps<{
  counter: number
  loading?: boolean
}>()
</script>

<template>
  <!--
    Systems still on the old my are a migration leftover: the card is only worth
    a grid slot while the company has some, so it stays hidden at zero (and
    while the count is still loading, to avoid a card that appears then leaves).
  -->
  <CounterCard
    v-if="!loading && counter > 0"
    :title="$t('systems.total_legacy_systems')"
    :counter="counter"
    :icon="faServer"
  >
    <template #title-tooltip>
      <i18n-t keypath="systems.total_legacy_systems_tooltip" tag="span" scope="global">
        <template #url>
          <NeLink :href="LEGACY_SYSTEMS_URL" target="_blank" rel="noopener noreferrer">
            {{ LEGACY_SYSTEMS_URL }}
          </NeLink>
        </template>
      </i18n-t>
    </template>
  </CounterCard>
</template>
