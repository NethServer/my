<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { faServer, faBuilding } from '@fortawesome/free-solid-svg-icons'
import PageBreadcrumb from '@/components/common/PageBreadcrumb.vue'
import { useResellerDetail } from '@/queries/organizations/resellerDetail'
import ResellerInfoCard from '@/components/resellers/ResellerInfoCard.vue'
import CounterCard from '@/components/common/CounterCard.vue'
import { useResellerStats } from '@/queries/organizations/resellerStats'
import { faGridOne } from '@nethesis/nethesis-solid-svg-icons'
import { useResellerSystems } from '@/queries/systems/resellerSystems'
import { useApplicationsSummaryByCompany } from '@/queries/applications/applicationsSummaryByCompany'
import OrganizationSystemsCard from '@/components/organizations/OrganizationSystemsCard.vue'
import OrganizationApplicationsCard from '@/components/organizations/OrganizationApplicationsCard.vue'
import { canReadResellers } from '@/lib/permissions'
import { computed } from 'vue'

const { state: resellerDetail } = useResellerDetail()
const { state: resellerStats } = useResellerStats()
const { state: resellerSystems } = useResellerSystems()
const { state: applicationsSummary } = useApplicationsSummaryByCompany()

// link to the Customers page filtered by this reseller as parent company.
// No include_hierarchy: the parent company filter matches exactly, so only the
// customers this reseller owns are listed.
const customersRoute = computed(() => {
  if (!resellerDetail.value.data) {
    return undefined
  }

  return {
    name: 'customers',
    query: {
      organization_id: resellerDetail.value.data.logto_id,
      organization_name: resellerDetail.value.data.name,
    },
  }
})

// link to the Systems page filtered by the whole reseller hierarchy
const hierarchySystemsRoute = computed(() => {
  if (!resellerDetail.value.data) {
    return undefined
  }

  return {
    name: 'systems',
    query: {
      organization_id: resellerDetail.value.data.logto_id,
      organization_name: resellerDetail.value.data.name,
      include_hierarchy: 'true',
    },
  }
})

// link to the Applications page filtered by the whole reseller hierarchy
const hierarchyApplicationsRoute = computed(() => {
  if (!resellerDetail.value.data) {
    return undefined
  }

  return {
    name: 'applications',
    query: {
      organization_id: resellerDetail.value.data.logto_id,
      organization_name: resellerDetail.value.data.name,
      include_hierarchy: 'true',
    },
  }
})
</script>

<template>
  <div>
    <PageBreadcrumb
      :section="$t('resellers.title')"
      :to="canReadResellers() ? '/resellers' : undefined"
      :current="resellerDetail.data?.name"
      :loading="resellerDetail.status === 'pending'"
    />
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
        :to="customersRoute"
      />
      <!-- total systems -->
      <CounterCard
        :title="$t('systems.total_systems')"
        :counter="resellerStats.data?.systems_hierarchy_count ?? 0"
        :icon="faServer"
        :loading="resellerStats.status === 'pending'"
        :to="hierarchySystemsRoute"
      />
      <!-- total applications -->
      <CounterCard
        :title="$t('applications.total_applications')"
        :counter="resellerStats.data?.applications_hierarchy_count ?? 0"
        :icon="faGridOne"
        :loading="resellerStats.status === 'pending'"
        :to="hierarchyApplicationsRoute"
      />
      <!-- organization systems -->
      <OrganizationSystemsCard
        :systems-count="resellerStats.data?.systems_count ?? 0"
        :systems-status="resellerSystems.status"
        :systems-data="resellerSystems.data"
        :stats-status="resellerStats.status"
        :organization-name="resellerDetail.data?.name"
      />
      <!-- organization applications -->
      <OrganizationApplicationsCard
        :applications-count="applicationsSummary.data?.total ?? 0"
        :applications-status="applicationsSummary.status"
        :summary-data="applicationsSummary.data"
        :organization-name="resellerDetail.data?.name"
      />
    </div>
  </div>
</template>
