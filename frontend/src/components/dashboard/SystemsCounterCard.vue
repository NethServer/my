<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faCircleCheck, faCircleXmark, faClock, faServer } from '@fortawesome/free-solid-svg-icons'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import CounterCard from '../common/CounterCard.vue'
import { NeBadgeV2 } from '@nethesis/vue-components'
import { getSystemsTotal, SYSTEMS_TOTAL_KEY } from '@/lib/systems/systems'
import { computed } from 'vue'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { abbreviateNumber } from '@/lib/common/index.ts'
import { useI18n } from 'vue-i18n'

const loginStore = useLoginStore()
const { locale } = useI18n()

const { state: systemsTotal } = useQuery({
  key: [SYSTEMS_TOTAL_KEY],
  enabled: () => !!loginStore.jwtToken,
  query: getSystemsTotal,
})

const totalCount = computed(() => systemsTotal.value.data?.total ?? 0)
const activeCount = computed(() => systemsTotal.value.data?.active ?? 0)
const inactiveCount = computed(() => systemsTotal.value.data?.inactive ?? 0)
const pendingCount = computed(() => systemsTotal.value.data?.unknown ?? 0)
</script>

<template>
  <CounterCard
    :title="$t('systems.total_systems')"
    :counter="totalCount"
    :icon="faServer"
    :loading="systemsTotal.status === 'pending'"
    title-route-name="systems"
  >
    <div class="mt-5 flex flex-wrap justify-center gap-2">
      <NeBadgeV2 v-if="activeCount > 0" kind="green">
        <FontAwesomeIcon :icon="faCircleCheck" class="size-4" />
        {{
          $t('systems.count_active', { count: abbreviateNumber(activeCount, locale) }, activeCount)
        }}
      </NeBadgeV2>
      <NeBadgeV2 v-if="inactiveCount > 0" kind="rose">
        <FontAwesomeIcon :icon="faCircleXmark" class="size-4" />
        {{
          $t(
            'systems.count_inactive',
            { count: abbreviateNumber(inactiveCount, locale) },
            inactiveCount,
          )
        }}
      </NeBadgeV2>
      <NeBadgeV2 v-if="pendingCount > 0" kind="gray">
        <FontAwesomeIcon :icon="faClock" class="size-4" />
        {{
          $t(
            'systems.count_pending',
            { count: abbreviateNumber(pendingCount, locale) },
            pendingCount,
          )
        }}
      </NeBadgeV2>
    </div>
  </CounterCard>
</template>
