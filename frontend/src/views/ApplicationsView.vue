<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import ApplicationsTable from '@/components/applications/ApplicationsTable.vue'
import {
  APPLICATIONS_TOTAL_KEY,
  getApplicationsTotal,
  saveShowUnassignedAppsNotificationToStorage,
  SHOW_UNASSIGNED_APPS_NOTIFICATION,
} from '@/lib/applications/applications'
import { useApplications } from '@/queries/applications'
import { useLoginStore } from '@/stores/login'
import { getPreference, NeHeading, NeInlineNotification } from '@nethesis/vue-components'
import { useQuery } from '@pinia/colada'
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loginStore = useLoginStore()

const { state: applicationsTotal } = useQuery({
  key: [APPLICATIONS_TOTAL_KEY],
  enabled: () => !!loginStore.jwtToken,
  query: getApplicationsTotal,
})

const { organizationFilter } = useApplications()

const justHiddenUnassignedAppsNotification = ref(false)

const showUnassignedAppsNotification = computed(() => {
  const username = loginStore.userInfo?.email

  if (!username || justHiddenUnassignedAppsNotification.value) {
    return false
  }

  let showNotificationFromPreference = getPreference(SHOW_UNASSIGNED_APPS_NOTIFICATION, username)

  if (showNotificationFromPreference === undefined) {
    // default to true if not set
    showNotificationFromPreference = true
  }

  return applicationsTotal.value.data?.unassigned && showNotificationFromPreference
})

const showUnassignedApps = () => {
  organizationFilter.value = ['no_org']
}

const dontShowUnassignedAppsNotificationAgain = () => {
  saveShowUnassignedAppsNotificationToStorage(false)
  justHiddenUnassignedAppsNotification.value = true
}
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('applications.title') }}</NeHeading>
    <div class="mb-8 max-w-2xl text-gray-500 dark:text-gray-400">
      {{ $t('applications.page_description') }}
    </div>
    <NeInlineNotification
      v-if="showUnassignedAppsNotification"
      kind="info"
      :description="
        $t('applications.num_applications_not_assigned', {
          num: applicationsTotal.data?.unassigned,
        })
      "
      :primary-button-label="t('applications.show_unassigned')"
      :secondary-button-label="t('applications.dont_show_again')"
      class="mb-8"
      @primary-click="showUnassignedApps"
      @secondary-click="dontShowUnassignedAppsNotificationAgain"
    />
    <!-- ////  -->
    <!-- showUnassignedAppsNotification {{ showUnassignedAppsNotification }}
    <NeButton @click="saveShowUnassignedAppsNotificationToStorage(true)"
      >Show Unassigned Apps</NeButton
    > -->
    <ApplicationsTable />
  </div>
</template>
