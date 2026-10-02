<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import ApplicationsTable from '@/components/applications/ApplicationsTable.vue'
import {
  getExport,
  saveShowUnassignedAppsNotificationToStorage,
  SHOW_UNASSIGNED_APPS_NOTIFICATION,
} from '@/lib/applications/applications'
import { downloadFile, exportFileName, getExportLimitError } from '@/lib/common'
import { useApplications } from '@/queries/applications/applications'
import { useApplicationsTotal } from '@/queries/applications/applicationsTotal'
import { useLoginStore } from '@/stores/login'
import { faChevronDown, faFileCsv, faFilePdf } from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  getPreference,
  NeButton,
  NeDropdown,
  NeHeading,
  NeInlineNotification,
} from '@nethesis/vue-components'
import { computed, ref } from 'vue'
import { useNotificationsStore } from '@/stores/notifications'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'

const { t } = useI18n()
const notificationsStore = useNotificationsStore()

// the export runs on the whole filtered list and can take a while: one at a time,
// with the Actions button showing progress
const isExporting = ref(false)
const route = useRoute()
const router = useRouter()
const loginStore = useLoginStore()

const { state: applicationsTotal } = useApplicationsTotal()

const {
  state,
  debouncedTextFilter,
  typeFilter,
  versionFilter,
  systemFilter,
  organizationFilter,
  parentOrganizationFilter,
  sortBy,
  sortDescending,
  clearFilters,
} = useApplications()

const justHiddenUnassignedAppsNotification = ref(false)

const showUnassignedAppsNotification = computed(() => {
  const username = loginStore.userInfo?.email

  // the count behind the notification ignores the Managed by filter, so it
  // would contradict the table while that filter is on
  if (
    !username ||
    justHiddenUnassignedAppsNotification.value ||
    parentOrganizationFilter.value.length
  ) {
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
  organizationFilter.value = [{ id: 'no_org', label: t('organizations.no_company') }]
}

// apply the filters requested via query params, then clean the URL
if (route.query.unassigned === 'true') {
  showUnassignedApps()
  router.replace({ query: {} })
}

// the filters render the label carried by the selection: the links pass the
// organization name, as it may not be among the options the dropdown loaded
const {
  organization_id: orgId,
  organization_name: orgName,
  parent_organization_id: parentOrgId,
  parent_organization_name: parentOrgName,
} = route.query

if (typeof orgId === 'string' && orgId && typeof orgName === 'string' && orgName) {
  clearFilters()
  organizationFilter.value = [{ id: orgId, label: orgName }]
  router.replace({ query: {} })
}

if (
  typeof parentOrgId === 'string' &&
  parentOrgId &&
  typeof parentOrgName === 'string' &&
  parentOrgName
) {
  clearFilters()
  parentOrganizationFilter.value = [{ id: parentOrgId, label: parentOrgName }]
  router.replace({ query: {} })
}

const dontShowUnassignedAppsNotificationAgain = () => {
  saveShowUnassignedAppsNotificationToStorage(false)
  justHiddenUnassignedAppsNotification.value = true
}

function getBulkActionsMenuItems() {
  return [
    {
      id: 'exportFilteredToPdf',
      label: t('applications.export_applications_to_pdf'),
      icon: faFilePdf,
      action: () => exportApplications('pdf'),
      disabled: isExporting.value || !state.value.data?.applications.length,
    },
    {
      id: 'exportFilteredToCsv',
      label: t('applications.export_applications_to_csv'),
      icon: faFileCsv,
      action: () => exportApplications('csv'),
      disabled: isExporting.value || !state.value.data?.applications.length,
    },
  ]
}

async function exportApplications(format: 'pdf' | 'csv') {
  try {
    isExporting.value = true
    const exportData = await getExport(
      format,
      debouncedTextFilter.value,
      typeFilter.value.map((o) => o.id),
      versionFilter.value.map((o) => o.id),
      systemFilter.value.map((o) => o.id),
      organizationFilter.value.map((o) => o.id),
      sortBy.value,
      sortDescending.value,
      parentOrganizationFilter.value.map((o) => o.id),
    )
    const fileName = exportFileName(t('applications.title'), format)
    downloadFile(exportData, fileName, format)
  } catch (error) {
    const limit = getExportLimitError(error)
    if (limit) {
      notificationsStore.createNotification({
        kind: 'error',
        title: t('common.too_many_records_to_export'),
        description: t('common.too_many_records_to_export_description', limit),
      })
      return
    }
    console.error(`Cannot export applications to ${format}:`, error)
    throw error
  } finally {
    isExporting.value = false
  }
}
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('applications.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('applications.page_description') }}
      </div>
      <div class="flex items-center gap-4">
        <NeDropdown
          :items="getBulkActionsMenuItems()"
          align-to-right
          :openMenuAriaLabel="$t('ne_dropdown.open_menu')"
        >
          <template #button>
            <NeButton :disabled="isExporting" :loading="isExporting" loading-position="suffix">
              <template #suffix>
                <FontAwesomeIcon :icon="faChevronDown" class="h-4 w-4" aria-hidden="true" />
              </template>
              {{ $t('common.actions') }}
            </NeButton>
          </template>
        </NeDropdown>
      </div>
    </div>
    <NeInlineNotification
      v-if="showUnassignedAppsNotification"
      kind="info"
      :description="
        $t('applications.num_applications_not_assigned', {
          count: applicationsTotal.data?.unassigned,
        })
      "
      :primary-button-label="t('applications.show_unassigned')"
      :secondary-button-label="t('applications.dont_show_again')"
      class="mb-8"
      @primary-click="showUnassignedApps"
      @secondary-click="dontShowUnassignedAppsNotificationAgain"
    />
    <ApplicationsTable />
  </div>
</template>
