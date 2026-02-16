<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowLeft } from '@fortawesome/free-solid-svg-icons'
import { useDistributorDetail } from '@/queries/organizations/distributorDetail'
import DistributorInfoCard from '@/components/distributors/DistributorInfoCard.vue'

const { state: distributorDetail } = useDistributorDetail()
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
      <DistributorInfoCard />
    </div>
  </div>
</template>
