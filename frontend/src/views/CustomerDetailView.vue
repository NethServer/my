<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowLeft } from '@fortawesome/free-solid-svg-icons'
import { useCustomerDetail } from '@/queries/organizations/customerDetail'
import CustomerInfoCard from '@/components/customers/CustomerInfoCard.vue'
import { useCustomerStats } from '@/queries/organizations/customerStats'
import { useCustomerSystems } from '@/queries/systems/customerSystems'
import OrganizationSystemsCard from '@/components/organizations/OrganizationSystemsCard.vue'
import OrganizationApplicationsCard from '@/components/organizations/OrganizationApplicationsCard.vue'

const { state: customerDetail } = useCustomerDetail()
const { state: customerStats } = useCustomerStats()
const { state: customerSystems } = useCustomerSystems()
</script>

<template>
  <div>
    <router-link to="/customers">
      <NeButton kind="tertiary" size="sm" class="mb-4 -ml-2">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowLeft" />
        </template>
        {{ $t('customers.title') }}
      </NeButton>
    </router-link>
    <!-- get customer detail error notification -->
    <NeInlineNotification
      v-if="customerDetail.status === 'error'"
      kind="error"
      :title="$t('customer_detail.cannot_retrieve_customer_detail')"
      :description="customerDetail.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="customerDetail.status === 'pending'" size="lg" class="mb-9 w-xs" />
    <NeHeading tag="h3" class="mb-7">
      {{ customerDetail.data?.name }}
    </NeHeading>
    <div class="3xl:grid-cols-4 grid grid-cols-1 gap-x-6 gap-y-6 md:grid-cols-2">
      <!-- customer info -->
      <CustomerInfoCard class="3xl:row-span-2 md:row-span-2" />
      <!-- organization systems -->
      <OrganizationSystemsCard
        :systems-count="customerStats.data?.systems_count ?? 0"
        :systems-status="customerSystems.status"
        :systems-data="customerSystems.data"
        :stats-status="customerStats.status"
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
