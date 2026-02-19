<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowLeft, faCity, faServer } from '@fortawesome/free-solid-svg-icons'
import { useDistributorDetail } from '@/queries/organizations/distributorDetail'
import DistributorInfoCard from '@/components/distributors/DistributorInfoCard.vue'
import CounterCard from '@/components/CounterCard.vue'
import { useDistributorStats } from '@/queries/organizations/distributorStats'
import { faGridOne } from '@nethesis/nethesis-solid-svg-icons'
import { useDistributorSystems } from '@/queries/systems/distributorSystems'
import { useApplicationsSummary } from '@/queries/applications/applicationsSummary'
import OrganizationSystemsCard from '@/components/organizations/OrganizationSystemsCard.vue'
import OrganizationApplicationsCard from '@/components/organizations/OrganizationApplicationsCard.vue'

const { state: distributorDetail } = useDistributorDetail()
const { state: distributorStats } = useDistributorStats()
const { state: distributorSystems } = useDistributorSystems()
const { state: applicationsSummary } = useApplicationsSummary()
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
      <OrganizationSystemsCard
        :systems-count="distributorStats.data?.systems_count ?? 0"
        :systems-status="distributorSystems.status"
        :systems-data="distributorSystems.data"
        :stats-status="distributorStats.status"
      />
      <!-- organization applications -->
      <OrganizationApplicationsCard
        :applications-count="applicationsSummary.data?.total ?? 0"
        :applications-status="applicationsSummary.status"
        :summary-data="applicationsSummary.data"
      />
    </div>
  </div>
</template>
