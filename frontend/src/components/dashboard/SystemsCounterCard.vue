<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { faCircleCheck, faCircleXmark, faClock, faServer } from '@fortawesome/free-solid-svg-icons'
import { useQuery } from '@pinia/colada'
import { useLoginStore } from '@/stores/login'
import CounterCard from '../common/CounterCard.vue'
import BadgeLink from '../common/BadgeLink.vue'
import { getSystemsTotal, SYSTEMS_TOTAL_KEY } from '@/lib/systems/systems'
import { computed } from 'vue'
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
    :to="{ name: 'systems' }"
  >
    <div class="mt-5 flex flex-wrap justify-center gap-2">
      <BadgeLink
        v-if="activeCount > 0"
        :to="{ name: 'systems', query: { status: 'active' } }"
        kind="green"
        :icon="faCircleCheck"
        :aria-label="$t('systems.show_active_systems')"
      >
        {{
          $t('systems.count_active', { count: abbreviateNumber(activeCount, locale) }, activeCount)
        }}
      </BadgeLink>
      <BadgeLink
        v-if="inactiveCount > 0"
        :to="{ name: 'systems', query: { status: 'inactive' } }"
        kind="rose"
        :icon="faCircleXmark"
        :aria-label="$t('systems.show_inactive_systems')"
      >
        {{
          $t(
            'systems.count_inactive',
            { count: abbreviateNumber(inactiveCount, locale) },
            inactiveCount,
          )
        }}
      </BadgeLink>
      <BadgeLink
        v-if="pendingCount > 0"
        :to="{ name: 'systems', query: { status: 'unknown' } }"
        kind="gray"
        :icon="faClock"
        :aria-label="$t('systems.show_pending_systems')"
      >
        {{
          $t(
            'systems.count_pending',
            { count: abbreviateNumber(pendingCount, locale) },
            pendingCount,
          )
        }}
      </BadgeLink>
    </div>
  </CounterCard>
</template>
