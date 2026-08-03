<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import PageBreadcrumb from '@/components/common/PageBreadcrumb.vue'
import { useApplicationDetail } from '@/queries/applications/applicationDetail'
import ApplicationInfoCard from '@/components/applications/ApplicationInfoCard.vue'
import ApplicationSystemCard from '@/components/applications/ApplicationSystemCard.vue'
import { getDisplayName } from '@/lib/applications/applications'
import { canReadApplications } from '@/lib/permissions'
import { computed } from 'vue'

const { state: applicationDetail } = useApplicationDetail()

const applicationName = computed(() => {
  if (!applicationDetail.value.data) {
    return '-'
  }
  return getDisplayName(applicationDetail.value.data)
})
</script>

<template>
  <div>
    <PageBreadcrumb
      :section="$t('applications.title')"
      :to="canReadApplications() ? '/applications' : undefined"
      :current="applicationName"
      :loading="applicationDetail.status === 'pending'"
    />
    <NeInlineNotification
      v-if="applicationDetail.status === 'error'"
      kind="error"
      :title="$t('application_detail.cannot_retrieve_application_detail')"
      :description="applicationDetail.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="applicationDetail.status === 'pending'" size="lg" class="mb-9 w-xs" />
    <NeHeading tag="h3" class="mb-7">
      {{ applicationName }}
    </NeHeading>
    <div class="3xl:grid-cols-4 grid grid-cols-1 gap-6 md:grid-cols-2">
      <ApplicationInfoCard />
      <ApplicationSystemCard />
    </div>
  </div>
</template>
