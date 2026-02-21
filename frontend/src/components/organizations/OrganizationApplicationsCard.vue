<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later
-->

<script setup lang="ts">
import { NeButton, NeLink } from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowRight } from '@fortawesome/free-solid-svg-icons'
import { faGridOne } from '@nethesis/nethesis-solid-svg-icons'
import CounterCard from '@/components/CounterCard.vue'
import { getApplicationLogo } from '@/lib/applications/applications'
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'
import router from '@/router'
import { useRoute } from 'vue-router'
import { useApplications } from '@/queries/applications/applications'
import { useApplicationsSummaryByCompany } from '@/queries/applications/applicationsSummaryByCompany'

const { t } = useI18n()
const route = useRoute()

const { state: applicationsSummary } = useApplicationsSummaryByCompany()
const { organizationFilter: organizationFilterForApps, clearFilters: clearApplicationsFilters } =
  useApplications()

const applicationsCount = computed(() => applicationsSummary.value?.data?.total ?? 0)
const moreApplications = computed(() => {
  if (!applicationsSummary.value.data) {
    return 0
  }
  const totalApps = applicationsSummary.value.data.total
  const retrievedApps = applicationsSummary.value.data.by_type.reduce(
    (acc, appType) => acc + appType.count,
    0,
  )
  const remainingApps = totalApps - retrievedApps

  if (remainingApps > 0) {
    return remainingApps
  }
  return 0
})

const goToApplications = () => {
  const companyId = route.params.companyId as string
  clearApplicationsFilters()
  organizationFilterForApps.value = companyId ? [companyId] : []
  router.push({ name: 'applications' })
}
</script>

<template>
  <CounterCard
    :title="$t('applications.organization_applications')"
    :counter="applicationsCount"
    :icon="faGridOne"
    :loading="applicationsSummary.status === 'pending'"
    :centeredCounter="!applicationsCount"
  >
    <div class="divide-y divide-gray-200 dark:divide-gray-700">
      <div
        v-for="appType in applicationsSummary.data?.by_type"
        :key="appType.instance_of"
        class="flex items-center justify-between py-3"
      >
        <div class="flex items-center gap-2">
          <img
            v-if="appType.instance_of"
            :src="getApplicationLogo(appType.instance_of)"
            :alt="appType.instance_of"
            aria-hidden="true"
            class="size-8"
          />
          <span class="font-medium">
            {{ appType.name || '-' }}
          </span>
        </div>
        <span>
          {{ appType.count }}
        </span>
      </div>
      <div v-if="moreApplications > 0" class="py-3">
        <NeLink @click="goToApplications()">
          {{ t('common.plus_n_more', { num: moreApplications }) }}
        </NeLink>
      </div>
    </div>
    <div v-if="applicationsCount > 0" class="flex justify-end">
      <NeButton kind="tertiary" class="mt-2" @click="goToApplications()">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowRight" aria-hidden="true" />
        </template>
        {{ t('common.go_to_page', { page: t('applications.title') }) }}
      </NeButton>
    </div>
  </CounterCard>
</template>
