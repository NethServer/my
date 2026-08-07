<!--
  Copyright (C) 2026 Nethesis S.r.l.
  SPDX-License-Identifier: GPL-3.0-or-later

  Add-ons tab of a system. Three endpoints back it — the grants, the add-ons
  the company is allowed to have, and the system's application instances —
  because the grant payload carries neither the add-on's name nor the name of
  the application it is scoped to. They are joined into rows here, once, and
  shared by the card grid and the detail table.

  The selected add-on lives in ?addon= next to ?tab=, so the drill-down
  survives a reload and answers the back button.
-->

<script setup lang="ts">
import { NeInlineNotification } from '@nethesis/vue-components'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { composeSystemAddonRows } from '@/lib/addons/systemAddons'
import {
  useAvailableAddons,
  useSystemAddons,
  useSystemApplications,
} from '@/queries/addons/systemAddons'
import { useSystemDetail } from '@/queries/systems/systemDetail'
import SystemAddonDetail from './addons/SystemAddonDetail.vue'
import SystemAddonsGrid from './addons/SystemAddonsGrid.vue'

const route = useRoute()
const router = useRouter()

const { state: addonsState, asyncStatus: addonsAsyncStatus } = useSystemAddons()
const { state: availableState } = useAvailableAddons()
const { state: applicationsState } = useSystemApplications()
const { state: systemDetail } = useSystemDetail()

const system = computed(() => systemDetail.value.data)

const selectedAddonId = computed(() => (route.query.addon as string) ?? '')

const isLoading = computed(
  () =>
    addonsState.value.status === 'pending' ||
    availableState.value.status === 'pending' ||
    applicationsState.value.status === 'pending',
)

const isRefreshing = computed(
  () => addonsAsyncStatus.value === 'loading' && addonsState.value.status !== 'pending',
)

const rows = computed(() =>
  composeSystemAddonRows({
    availableAddons: availableState.value.data ?? [],
    grants: addonsState.value.data ?? [],
    applications: applicationsState.value.data?.applications ?? [],
    systemType: system.value?.type ?? '',
    systemName: system.value?.name ?? '',
  }),
)

const selectedRows = computed(() =>
  rows.value.filter((row) => row.addon.id === selectedAddonId.value),
)

// Losing the grants is fatal to the page; losing availability only costs the
// add-ons nobody bought yet, so that one warns and lets the rest render.
const isGridShown = computed(() => addonsState.value.status !== 'error')

function openAddon(addonId: string) {
  router.push({ query: { ...route.query, addon: addonId } })
}

function closeAddon() {
  const { addon, ...query } = route.query
  void addon
  router.push({ query })
}
</script>

<template>
  <div>
    <!-- page description -->
    <div class="text-tertiary-neutral mb-8 max-w-3xl">
      {{ $t('addons.system_addons_description') }}
    </div>
    <!-- get add-ons error notification -->
    <NeInlineNotification
      v-if="addonsState.status === 'error'"
      kind="error"
      :title="$t('addons.cannot_retrieve_system_addons')"
      :description="addonsState.error.message"
      class="mb-6"
    />
    <!-- get available add-ons error notification -->
    <NeInlineNotification
      v-if="availableState.status === 'error'"
      kind="warning"
      :title="$t('addons.cannot_retrieve_available_addons')"
      :description="availableState.error.message"
      class="mb-6"
    />
    <!-- get applications error notification -->
    <NeInlineNotification
      v-if="applicationsState.status === 'error'"
      kind="warning"
      :title="$t('addons.cannot_retrieve_applications')"
      :description="applicationsState.error.message"
      class="mb-6"
    />
    <template v-if="isGridShown">
      <SystemAddonDetail
        v-if="selectedAddonId"
        :addon-id="selectedAddonId"
        :rows="selectedRows"
        :loading="isLoading"
        :refreshing="isRefreshing"
        @back="closeAddon"
      />
      <SystemAddonsGrid
        v-else
        :rows="rows"
        :loading="isLoading"
        :refreshing="isRefreshing"
        @open="openAddon"
      />
    </template>
  </div>
</template>
