<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  How many organizations have rebranding configured. A counter with a per-type
  split, which is exactly CounterCard plus badges — so it borrows both rather
  than drawing anything of its own.
-->

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { faPalette } from '@fortawesome/free-solid-svg-icons'
import { abbreviateNumber } from '@/lib/common'
import { useRebrandingSummary } from '@/queries/dashboard/rebrandingSummary'
import CounterCard from '@/components/common/CounterCard.vue'
import BadgeLink from '@/components/common/BadgeLink.vue'

const { locale } = useI18n()
const { state } = useRebrandingSummary()

const summary = computed(() => state.value?.data)
const isLoading = computed(() => state.value?.status === 'pending')

const distributors = computed(() => summary.value?.distributors ?? 0)
const resellers = computed(() => summary.value?.resellers ?? 0)
const customers = computed(() => summary.value?.customers ?? 0)
</script>

<template>
  <CounterCard
    :title="$t('dashboard.widget_rebranding_summary')"
    :counter="summary?.total ?? 0"
    :icon="faPalette"
    :loading="isLoading"
  >
    <div class="mt-5 flex flex-wrap justify-center gap-2">
      <BadgeLink v-if="distributors > 0" :to="{ name: 'distributors' }" kind="blue">
        {{
          $t(
            'dashboard.count_distributors',
            { count: abbreviateNumber(distributors, locale) },
            distributors,
          )
        }}
      </BadgeLink>
      <BadgeLink v-if="resellers > 0" :to="{ name: 'resellers' }" kind="indigo">
        {{
          $t('dashboard.count_resellers', { count: abbreviateNumber(resellers, locale) }, resellers)
        }}
      </BadgeLink>
      <BadgeLink v-if="customers > 0" :to="{ name: 'customers' }" kind="gray">
        {{
          $t('dashboard.count_customers', { count: abbreviateNumber(customers, locale) }, customers)
        }}
      </BadgeLink>
    </div>
  </CounterCard>
</template>
