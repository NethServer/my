<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeHeading, NeInlineNotification, NeSkeleton } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowLeft } from '@fortawesome/free-solid-svg-icons'
import { useApplicationDetail } from '@/queries/applications/applicationDetail'
import ApplicationInfoCard from '@/components/applications/ApplicationInfoCard.vue'
import ApplicationSystemCard from '@/components/applications/ApplicationSystemCard.vue'
import { getDisplayName } from '@/lib/applications/applications'
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
    <router-link to="/applications">
      <NeButton kind="tertiary" size="sm" class="mb-4 -ml-2">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowLeft" />
        </template>
        {{ $t('applications.title') }}
      </NeButton>
    </router-link>
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
