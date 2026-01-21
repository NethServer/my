<!--
  Copyright (C) 2025 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import ApplicationsTable from '@/components/applications/ApplicationsTable.vue'
import { NeHeading } from '@nethesis/vue-components'
import { useI18n } from 'vue-i18n'

//// review (search "system")

const { t } = useI18n()

// const { ////
//   state,
//   debouncedTextFilter,
//   productFilter,
//   createdByFilter,
//   versionFilter,
//   statusFilter,
//   sortBy,
//   sortDescending,
// } = useApplications()

// const applicationsPage = computed(() => {
//   return state.value.data?.applications || []
// })
</script>

<template>
  <div>
    <NeHeading tag="h3" class="mb-7">{{ $t('applications.title') }}</NeHeading>
    <div class="mb-8 flex flex-col items-start justify-between gap-6 xl:flex-row">
      <div class="max-w-2xl text-gray-500 dark:text-gray-400">
        {{ $t('applications.page_description') }}
      </div>
    </div>
    <!-- //// todo applications without org notification -->
    <ApplicationsTable />
  </div>
</template>
