<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import {
  NeButton,
  NeHeading,
  NeInlineNotification,
  NeLink,
  NeSkeleton,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowLeft, faArrowRight, faCity, faServer } from '@fortawesome/free-solid-svg-icons'
import { useDistributorDetail } from '@/queries/organizations/distributorDetail'
import DistributorInfoCard from '@/components/distributors/DistributorInfoCard.vue'
import CounterCard from '@/components/CounterCard.vue'
import { useDistributorStats } from '@/queries/organizations/distributorStats'
import { faGridOne } from '@nethesis/nethesis-solid-svg-icons'
import { useDistributorSystems } from '@/queries/systems/distributorSystems'
import { getProductLogo, getProductName } from '@/lib/systems/systems'
import SystemStatusIcon from '@/components/systems/SystemStatusIcon.vue'
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'
import router from '@/router'
import { useRoute } from 'vue-router'
import { useSystems } from '@/queries/systems/systems'

const { t } = useI18n()
const route = useRoute()
const { state: distributorDetail } = useDistributorDetail()
const { state: distributorStats } = useDistributorStats()
const { state: distributorSystems } = useDistributorSystems()
const { organizationFilter: organizationFilterForSystems } = useSystems()
// const { organizationFilter: organizationFilterForApps } = useApplications() ////

const moreSystems = computed(() => {
  if (!distributorSystems.value.data) {
    return 0
  }
  const totalSystems = distributorStats.value.data?.systems_count ?? 0
  const retrievedSystems = distributorSystems.value.data.systems.length
  const remainingSystems = totalSystems - retrievedSystems

  if (remainingSystems > 0) {
    return remainingSystems
  }
  return 0
})

const goToSystems = () => {
  const distributorId = route.params.distributorId as string
  organizationFilterForSystems.value = distributorId ? [distributorId] : []
  router.push({ name: 'systems' })
}

// const goToApplications = () => { ////
//   const distributorId = route.params.distributorId as string
//   organizationFilterForApps.value = distributorId ? [distributorId] : []
//   router.push({ name: 'applications' })
// }
</script>

<template>
  <div>
    <router-link to="/distributors">
      <NeButton kind="tertiary" size="sm" class="mb-4 -ml-2">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowLeft" />
        </template>
        {{ $t('distributors.title') }}
      </NeButton>
    </router-link>
    <!-- get distributor detail error notification -->
    <NeInlineNotification
      v-if="distributorDetail.status === 'error'"
      kind="error"
      :title="$t('distributor_detail.cannot_retrieve_distributor_detail')"
      :description="distributorDetail.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="distributorDetail.status === 'pending'" size="lg" class="mb-9 w-xs" />
    <NeHeading tag="h3" class="mb-7">
      {{ distributorDetail.data?.name }}
    </NeHeading>
    <div class="3xl:grid-cols-4 grid grid-cols-1 gap-x-6 gap-y-6 md:grid-cols-2">
      <!-- distributor info -->
      <DistributorInfoCard class="3xl:row-span-2 md:row-span-3" />
      <!-- resellers -->
      <CounterCard
        :title="$t('resellers.title')"
        :counter="distributorStats.data?.resellers_count ?? 0"
        :icon="faCity"
        :loading="distributorStats.status === 'pending'"
      />
      <!-- total systems -->
      <CounterCard
        :title="$t('systems.total_systems')"
        :counter="distributorStats.data?.systems_hierarchy_count ?? 0"
        :icon="faServer"
        :loading="distributorStats.status === 'pending'"
      />
      <!-- total applications -->
      <CounterCard
        :title="$t('applications.total_applications')"
        :counter="distributorStats.data?.applications_hierarchy_count ?? 0"
        :icon="faGridOne"
        :loading="distributorStats.status === 'pending'"
      />
      <!-- organization systems -->
      <!-- //// externalize component -->
      <CounterCard
        :title="$t('systems.organization_systems')"
        :counter="distributorStats.data?.systems_count ?? 0"
        :icon="faServer"
        :loading="distributorStats.status === 'pending' || distributorSystems.status === 'pending'"
        :centeredCounter="false"
      >
        <div class="divide-y divide-gray-200 dark:divide-gray-700">
          <div
            v-for="system in distributorSystems.data?.systems"
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
        <div class="flex justify-end">
          <NeButton kind="tertiary" class="mt-2" @click="goToSystems()">
            <template #prefix>
              <FontAwesomeIcon :icon="faArrowRight" aria-hidden="true" />
            </template>
            {{ t('common.go_to_page', { page: t('systems.title') }) }}
          </NeButton>
        </div>
      </CounterCard>
      <!-- organization applications -->
      <CounterCard
        :title="$t('applications.organization_applications')"
        :counter="distributorStats.data?.applications_count ?? 0"
        :icon="faGridOne"
        :loading="distributorStats.status === 'pending'"
        :centeredCounter="false"
      />
    </div>
  </div>
</template>
