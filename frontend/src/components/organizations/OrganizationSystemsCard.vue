<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeLink } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowRight, faServer } from '@fortawesome/free-solid-svg-icons'
import CounterCard from '@/components/CounterCard.vue'
import { getProductLogo, getProductName } from '@/lib/systems/systems'
import SystemStatusIcon from '@/components/systems/SystemStatusIcon.vue'
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'
import router from '@/router'
import { useRoute } from 'vue-router'
import { useSystems } from '@/queries/systems/systems'
import type { System } from '@/lib/systems/systems'

interface SystemData {
  systems: System[] | null
}

const props = defineProps<{
  systemsCount: number
  systemsStatus: 'pending' | 'success' | 'error'
  systemsData: SystemData | undefined
  statsStatus: 'pending' | 'success' | 'error'
}>()

const { t } = useI18n()
const route = useRoute()
const { organizationFilter: organizationFilterForSystems } = useSystems()

const moreSystems = computed(() => {
  if (!props.systemsData) {
    return 0
  }
  const totalSystems = props.systemsCount ?? 0
  const retrievedSystems = props.systemsData.systems?.length ?? 0
  const remainingSystems = totalSystems - retrievedSystems

  if (remainingSystems > 0) {
    return remainingSystems
  }
  return 0
})

const goToSystems = () => {
  const companyId = route.params.companyId as string
  organizationFilterForSystems.value = companyId ? [companyId] : []
  router.push({ name: 'systems' })
}
</script>

<template>
  <CounterCard
    :title="$t('systems.organization_systems')"
    :counter="systemsCount"
    :icon="faServer"
    :loading="statsStatus === 'pending' || systemsStatus === 'pending'"
    :centeredCounter="!systemsCount"
  >
    <div class="divide-y divide-gray-200 dark:divide-gray-700">
      <div
        v-for="system in systemsData?.systems"
        :key="system.id"
        class="flex items-center justify-between gap-4 py-3"
      >
        <router-link
          :to="{ name: 'system_detail', params: { systemId: system.id } }"
          class="cursor-pointer font-medium hover:underline"
        >
          <div class="flex items-center gap-2">
            <img
              v-if="system.type"
              :src="getProductLogo(system.type)"
              :alt="getProductName(system.type)"
              aria-hidden="true"
              class="size-8"
            />
            <span>
              {{ system.name || '-' }}
            </span>
          </div>
        </router-link>
        <div class="flex items-center gap-2">
          <SystemStatusIcon :status="system.status" :suspended-at="system.suspended_at" />
          <span v-if="system.suspended_at">
            {{ t('common.suspended') }}
          </span>
          <span v-else-if="system.status">
            {{ t(`systems.status_${system.status}`) }}
          </span>
          <span v-else>-</span>
        </div>
      </div>
      <div v-if="moreSystems > 0" class="py-3">
        <NeLink @click="goToSystems()">
          {{ t('common.plus_n_more', { num: moreSystems }) }}
        </NeLink>
      </div>
    </div>
    <div v-if="systemsCount > 0" class="flex justify-end">
      <NeButton kind="tertiary" class="mt-2" @click="goToSystems()">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowRight" aria-hidden="true" />
        </template>
        {{ t('common.go_to_page', { page: t('systems.title') }) }}
      </NeButton>
    </div>
  </CounterCard>
</template>
