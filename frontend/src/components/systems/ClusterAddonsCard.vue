<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Overview of a cluster's add-ons. A NethServer inventory reports no add-on
  features the way a NethSecurity one does, so this card reads the same grants
  as the add-ons tab instead, and names each add-on with the status the tab
  shows for it.
-->

<script setup lang="ts">
import {
  NeButton,
  NeCard,
  NeEmptyState,
  NeHeading,
  NeInlineNotification,
  NeSkeleton,
} from '@nethesis/vue-components'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faArrowRight, faPuzzlePiece } from '@fortawesome/free-solid-svg-icons'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import router from '@/router'
import AddonStatusIcon from '@/components/systems/addons/AddonStatusIcon.vue'
import { ADDON_ROW_STATUSES, composeSystemAddonRows, getRowStatus } from '@/lib/addons/systemAddons'
import {
  useAvailableAddons,
  useSystemAddons,
  useSystemApplications,
} from '@/queries/addons/systemAddons'
import { useSystemDetail } from '@/queries/systems/systemDetail'

const { t } = useI18n()
const route = useRoute()

const { state: addonsState } = useSystemAddons()
const { state: availableState } = useAvailableAddons()
const { state: applicationsState } = useSystemApplications()
const { state: systemDetail } = useSystemDetail()

const isLoading = computed(
  () =>
    addonsState.value.status === 'pending' ||
    availableState.value.status === 'pending' ||
    applicationsState.value.status === 'pending',
)

// Held add-ons first, in the order the tab lists statuses, and the ones the
// cluster could still buy last.
const rows = computed(() =>
  composeSystemAddonRows({
    availableAddons: availableState.value.data ?? [],
    grants: addonsState.value.data ?? [],
    applications: applicationsState.value.data?.applications ?? [],
    systemType: systemDetail.value.data?.type ?? '',
    systemName: systemDetail.value.data?.name ?? '',
  }).sort(
    (a, b) =>
      ADDON_ROW_STATUSES.indexOf(getRowStatus(a)) - ADDON_ROW_STATUSES.indexOf(getRowStatus(b)),
  ),
)

const goToAddons = () => {
  router.push({
    name: 'system_detail',
    params: { systemId: route.params.systemId },
    query: { ...route.query, tab: 'addons' },
  })
}
</script>

<template>
  <NeCard>
    <div class="mb-4 flex h-10 items-center gap-4">
      <FontAwesomeIcon :icon="faPuzzlePiece" class="size-5 shrink-0" aria-hidden="true" />
      <NeHeading tag="h6">
        {{ $t('system_detail.addons').toUpperCase() }}
      </NeHeading>
    </div>
    <!-- error -->
    <NeInlineNotification
      v-if="addonsState.status === 'error'"
      kind="error"
      :title="$t('addons.cannot_retrieve_system_addons')"
      :description="addonsState.error.message"
      class="mb-6"
    />
    <NeSkeleton v-else-if="isLoading" :lines="8" />
    <div v-else-if="rows.length > 0" class="divide-y divide-gray-200 dark:divide-gray-700">
      <div
        v-for="row in rows"
        :key="row.addon.id"
        class="flex items-center justify-between gap-2 py-4"
      >
        <span class="text-primary-neutral font-medium">
          {{ row.addon.display_name }}
        </span>
        <AddonStatusIcon :status="getRowStatus(row)" class="text-tertiary-neutral font-medium" />
      </div>
    </div>
    <NeEmptyState v-else :title="$t('system_detail.no_addons')" :icon="faPuzzlePiece" />
    <div v-if="!isLoading" class="flex justify-end">
      <NeButton kind="tertiary" class="mt-2" @click="goToAddons()">
        <template #prefix>
          <FontAwesomeIcon :icon="faArrowRight" aria-hidden="true" />
        </template>
        {{ t('common.go_to_page', { page: t('addons.title') }) }}
      </NeButton>
    </div>
  </NeCard>
</template>
