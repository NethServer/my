<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeBadgeV2, NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import PageBreadcrumb from '@/components/common/PageBreadcrumb.vue'
import { useIsOwnCompany } from '@/composables/useIsOwnCompany'
import { useCustomerDetail } from '@/queries/organizations/customerDetail'
import CustomerInfoCard from '@/components/customers/CustomerInfoCard.vue'
import OrganizationContactsCard from '@/components/organizations/OrganizationContactsCard.vue'
import { useCustomerStats } from '@/queries/organizations/customerStats'
import { useCustomerSystems } from '@/queries/systems/customerSystems'
import OrganizationSystemsCard from '@/components/organizations/OrganizationSystemsCard.vue'
import OrganizationApplicationsCard from '@/components/organizations/OrganizationApplicationsCard.vue'
import { canReadCustomers } from '@/lib/permissions'

const { state: customerDetail } = useCustomerDetail()
const isOwnCompany = useIsOwnCompany()
const { state: customerStats } = useCustomerStats()
const { state: customerSystems } = useCustomerSystems()
</script>

<template>
  <div>
    <!-- no list to go back to from the user's own company -->
    <PageBreadcrumb
      v-if="!isOwnCompany"
      :section="$t('customers.title')"
      :to="canReadCustomers() ? '/customers' : undefined"
      :current="customerDetail.data?.name"
      :loading="customerDetail.status === 'pending'"
    />
    <!-- get customer detail error notification -->
    <NeInlineNotification
      v-if="customerDetail.status === 'error'"
      kind="error"
      :title="$t('customer_detail.cannot_retrieve_customer_detail')"
      :description="customerDetail.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="customerDetail.status === 'pending'" size="lg" class="mb-9 w-xs" />
    <div class="mb-7 flex flex-wrap items-center gap-4">
      <NeHeading tag="h3">
        {{ customerDetail.data?.name }}
      </NeHeading>
      <NeBadgeV2 v-if="isOwnCompany && customerDetail.data" kind="indigo">
        {{ $t('organizations.your_company') }}
      </NeBadgeV2>
    </div>
    <div class="3xl:grid-cols-4 grid grid-cols-1 gap-x-6 gap-y-6 md:grid-cols-2">
      <!-- customer info -->
      <CustomerInfoCard class="row-span-4" />
      <!-- customer contacts -->
      <OrganizationContactsCard
        :contacts="customerDetail.data?.custom_data"
        :loading="customerDetail.status === 'pending'"
        class="row-span-4"
      />
      <!-- organization systems -->
      <OrganizationSystemsCard
        :systems-count="customerStats.data?.systems_count ?? 0"
        :systems-status="customerSystems.status"
        :systems-data="customerSystems.data"
        :stats-status="customerStats.status"
        :organization-name="customerDetail.data?.name"
      />
      <!-- organization applications -->
      <OrganizationApplicationsCard :organization-name="customerDetail.data?.name" />
    </div>
  </div>
</template>
