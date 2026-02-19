<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowLeft, faServer, faBuilding } from '@fortawesome/free-solid-svg-icons'
import { useResellerDetail } from '@/queries/organizations/resellerDetail'
import ResellerInfoCard from '@/components/resellers/ResellerInfoCard.vue'
import CounterCard from '@/components/CounterCard.vue'
import { useResellerStats } from '@/queries/organizations/resellerStats'
import { faGridOne } from '@nethesis/nethesis-solid-svg-icons'
import { useResellerSystems } from '@/queries/systems/resellerSystems'
import { useApplicationsSummary } from '@/queries/applications/applicationsSummary'
import OrganizationSystemsCard from '@/components/organizations/OrganizationSystemsCard.vue'
import OrganizationApplicationsCard from '@/components/organizations/OrganizationApplicationsCard.vue'

const { state: resellerDetail } = useResellerDetail()
const { state: resellerStats } = useResellerStats()
const { state: resellerSystems } = useResellerSystems()
const { state: applicationsSummary } = useApplicationsSummary()
</script>

<template>
  <div>
    <router-link to="/resellers">
      <NeButton kind="tertiary" size="sm" class="mb-4 -ml-2">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowLeft" />
        </template>
        {{ $t('resellers.title') }}
      </NeButton>
    </router-link>
    <!-- get reseller detail error notification -->
    <NeInlineNotification
      v-if="resellerDetail.status === 'error'"
      kind="error"
      :title="$t('reseller_detail.cannot_retrieve_reseller_detail')"
      :description="resellerDetail.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="resellerDetail.status === 'pending'" size="lg" class="mb-9 w-xs" />
    <NeHeading tag="h3" class="mb-7">
      {{ resellerDetail.data?.name }}
    </NeHeading>
    <div class="3xl:grid-cols-4 grid grid-cols-1 gap-x-6 gap-y-6 md:grid-cols-2">
      <!-- reseller info -->
      <ResellerInfoCard class="3xl:row-span-2 md:row-span-3" />
      <!-- customers -->
      <CounterCard
        :title="$t('customers.title')"
        :counter="resellerStats.data?.customers_count ?? 0"
        :icon="faBuilding"
        :loading="resellerStats.status === 'pending'"
      />
      <!-- total systems -->
      <CounterCard
        :title="$t('systems.total_systems')"
        :counter="resellerStats.data?.systems_hierarchy_count ?? 0"
        :icon="faServer"
        :loading="resellerStats.status === 'pending'"
      />
      <!-- total applications -->
      <CounterCard
        :title="$t('applications.total_applications')"
        :counter="resellerStats.data?.applications_hierarchy_count ?? 0"
        :icon="faGridOne"
        :loading="resellerStats.status === 'pending'"
      />
      <!-- organization systems -->
      <OrganizationSystemsCard
        :systems-count="resellerStats.data?.systems_count ?? 0"
        :systems-status="resellerSystems.status"
        :systems-data="resellerSystems.data"
        :stats-status="resellerStats.status"
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
